package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/misbakhul29/goks/pkg/auth"
	"github.com/misbakhul29/goks/pkg/router"
)

func TestNewJWT_SecretLength(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic on short JWT secret")
		}
	}()
	auth.NewJWT("too-short-secret", time.Hour)
}

func TestNewJWT_SignAndVerify(t *testing.T) {
	secret := "a-very-long-secret-key-that-is-at-least-32-chars-long"
	j := auth.NewJWT(secret, time.Hour)

	user := &auth.User{ID: 123, Email: "admin@example.com", Roles: []string{"admin"}}
	token, err := j.Sign(user)
	if err != nil {
		t.Fatalf("unexpected error signing token: %v", err)
	}

	claims, err := j.Verify(token)
	if err != nil {
		t.Fatalf("unexpected error verifying token: %v", err)
	}

	if claims.Sub != 123 {
		t.Fatalf("expected sub 123, got %d", claims.Sub)
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "admin" {
		t.Fatalf("expected role admin, got %v", claims.Roles)
	}
}

func TestManager_Login_SecureCookie(t *testing.T) {
	m := auth.New(time.Hour)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "https://example.com/login", nil)
	req.Header.Set("X-Forwarded-Proto", "https")

	user := &auth.User{ID: 456, Email: "user@example.com"}
	_, err := m.Login(rec, req, user)
	if err != nil {
		t.Fatalf("unexpected login error: %v", err)
	}

	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "goks_session" {
			sessionCookie = c
			break
		}
	}

	if sessionCookie == nil {
		t.Fatalf("expected goks_session cookie to be set")
	}
	if !sessionCookie.Secure {
		t.Fatalf("expected session cookie to have Secure=true over HTTPS")
	}
	if !sessionCookie.HttpOnly {
		t.Fatalf("expected session cookie to have HttpOnly=true")
	}
}

func TestManager_SessionFixationProtection(t *testing.T) {
	m := auth.New(time.Hour)
	defer m.Close()

	// 1. First login
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest("POST", "/login", nil)
	sess1, err := m.Login(w1, r1, &auth.User{ID: 1, Email: "user1@example.com"})
	if err != nil {
		t.Fatalf("first login failed: %v", err)
	}

	// 2. Second login using same cookie (attacker/fixation scenario)
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("POST", "/login", nil)
	r2.AddCookie(&http.Cookie{Name: "goks_session", Value: sess1.Token})

	sess2, err := m.Login(w2, r2, &auth.User{ID: 2, Email: "user2@example.com"})
	if err != nil {
		t.Fatalf("second login failed: %v", err)
	}

	// Old session token MUST be invalidated
	if _, ok := m.SessionFromRequest(r2); ok {
		t.Fatal("expected old session token to be invalidated upon new login (session fixation prevention)")
	}

	// New session must be distinct and valid
	if sess1.Token == sess2.Token {
		t.Fatal("expected new session token to differ from old token")
	}
}

func TestManager_RememberMe(t *testing.T) {
	m := auth.New(time.Hour)
	defer m.Close()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/login", nil)

	user := &auth.User{ID: 10, Email: "remember@example.com"}
	sess, err := m.Login(rec, req, user, auth.WithRememberMe(true))
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if sess.ExpiresAt.Before(time.Now().Add(29 * 24 * time.Hour)) {
		t.Fatalf("expected remember me session to expire in ~30 days, got %v", sess.ExpiresAt)
	}

	var found *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "goks_session" {
			found = c
			break
		}
	}
	if found == nil || found.MaxAge <= 0 {
		t.Fatalf("expected cookie MaxAge to be set for remember me")
	}
}

func TestPasswordHashing(t *testing.T) {
	password := "SecretP@ssw0rd!"
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if hash == password {
		t.Fatalf("expected hashed password to differ from raw password")
	}

	if !auth.CheckPassword(hash, password) {
		t.Fatalf("CheckPassword expected true for correct password")
	}

	if auth.CheckPassword(hash, "wrong-password") {
		t.Fatalf("CheckPassword expected false for wrong password")
	}
}

func TestFlashMessages(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	auth.SetFlash(w, r, "success", "Item created successfully")
	res := w.Result()
	cookies := res.Cookies()

	var flashCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "goks_flash_success" {
			flashCookie = c
			break
		}
	}
	if flashCookie == nil {
		t.Fatalf("expected flash cookie to be set")
	}

	// Simulate subsequent request reading the flash cookie
	r2 := httptest.NewRequest("GET", "/next", nil)
	r2.AddCookie(flashCookie)
	w2 := httptest.NewRecorder()

	msg := auth.GetFlash(w2, r2, "success")
	if msg != "Item created successfully" {
		t.Fatalf("expected 'Item created successfully', got %q", msg)
	}

	// Verify cookie is cleared
	w2Cookies := w2.Result().Cookies()
	var cleared *http.Cookie
	for _, c := range w2Cookies {
		if c.Name == "goks_flash_success" {
			cleared = c
			break
		}
	}
	if cleared == nil || cleared.MaxAge != -1 {
		t.Fatalf("expected flash cookie to be cleared on read")
	}
}

func TestIntendedURL(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/admin/settings", nil)

	auth.SetIntendedURL(w, r, "/admin/settings")
	cookies := w.Result().Cookies()

	var cookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "goks_intended_url" {
			cookie = c
			break
		}
	}
	if cookie == nil {
		t.Fatalf("expected intended URL cookie")
	}

	// Read intended URL
	r2 := httptest.NewRequest("GET", "/login", nil)
	r2.AddCookie(cookie)
	w2 := httptest.NewRecorder()

	target := auth.GetIntendedURL(w2, r2, "/dashboard")
	if target != "/admin/settings" {
		t.Fatalf("expected /admin/settings, got %q", target)
	}

	// Test security: open redirect attempts must fallback to default
	w3 := httptest.NewRecorder()
	r3 := httptest.NewRequest("GET", "/", nil)
	auth.SetIntendedURL(w3, r3, "https://malicious.com")
	if len(w3.Result().Cookies()) > 0 {
		// Even if malicious cookie is sent, GetIntendedURL rejects external hosts
		rMal := httptest.NewRequest("GET", "/", nil)
		rMal.AddCookie(w3.Result().Cookies()[0])
		safe := auth.GetIntendedURL(httptest.NewRecorder(), rMal, "/fallback")
		if safe != "/fallback" {
			t.Fatalf("expected external URL to be rejected, got %s", safe)
		}
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := auth.NewRateLimiter(3, 500*time.Millisecond)
	key := "192.168.1.1"

	if limiter.TooManyAttempts(key) {
		t.Fatalf("expected limiter not to block initially")
	}

	limiter.Hit(key)
	limiter.Hit(key)
	if limiter.TooManyAttempts(key) {
		t.Fatalf("expected limiter not to block on 2 attempts")
	}

	limiter.Hit(key)
	if !limiter.TooManyAttempts(key) {
		t.Fatalf("expected limiter to block after 3 attempts")
	}

	limiter.Clear(key)
	if limiter.TooManyAttempts(key) {
		t.Fatalf("expected limiter to reset after Clear")
	}
}

func TestHooks(t *testing.T) {
	loginTriggered := false
	logoutTriggered := false

	auth.OnLogin(func(ctx context.Context, user *auth.User, r *http.Request) {
		if user != nil && user.Email == "hook@example.com" {
			loginTriggered = true
		}
	})

	auth.OnLogout(func(ctx context.Context, user *auth.User, r *http.Request) {
		if user != nil && user.Email == "hook@example.com" {
			logoutTriggered = true
		}
	})

	m := auth.New(time.Hour)
	defer m.Close()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/login", nil)
	user := &auth.User{ID: 99, Email: "hook@example.com"}

	sess, err := m.Login(rec, req, user)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if !loginTriggered {
		t.Fatalf("expected OnLogin hook to be triggered")
	}

	req2 := httptest.NewRequest("POST", "/logout", nil)
	req2.AddCookie(&http.Cookie{Name: "goks_session", Value: sess.Token})
	rec2 := httptest.NewRecorder()

	m.Logout(rec2, req2)
	if !logoutTriggered {
		t.Fatalf("expected OnLogout hook to be triggered")
	}
}

func TestGuestMiddleware(t *testing.T) {
	mw := auth.Guest("/custom-dashboard")
	handlerCalled := false
	handler := mw(func(ctx *router.Context) error {
		handlerCalled = true
		return ctx.Text("ok")
	})

	// 1. Unauthenticated request should proceed
	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/login", nil)
	ctx1 := router.NewTestContext(rec1, req1)
	_ = handler(ctx1)
	if !handlerCalled {
		t.Fatalf("expected unauthenticated request to reach handler")
	}

	m := auth.Default
	user := &auth.User{ID: 7, Email: "guest@example.com"}
	wLog := httptest.NewRecorder()
	sess, _ := m.Login(wLog, req1, user)

	// 2. Authenticated request to guest route should redirect
	reqAuth := httptest.NewRequest("GET", "/login", nil)
	reqAuth.AddCookie(&http.Cookie{Name: "goks_session", Value: sess.Token})
	recAuth := httptest.NewRecorder()

	ctx2 := router.NewTestContext(recAuth, reqAuth)
	_ = handler(ctx2)

	if recAuth.Code != http.StatusFound {
		t.Fatalf("expected status 302 redirect for guest, got %d", recAuth.Code)
	}
	if loc := recAuth.Header().Get("Location"); loc != "/custom-dashboard" {
		t.Fatalf("expected redirect to /custom-dashboard, got %q", loc)
	}
}

func TestFlash_NilWriter(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	// Must not panic when w is nil
	auth.SetFlash(nil, req, "key", "val")
	val := auth.GetFlash(nil, req, "key")
	if val != "" {
		t.Fatalf("expected empty flash, got %q", val)
	}

	auth.SetIntendedURL(nil, req, "/admin")
	target := auth.GetIntendedURL(nil, req, "/fallback")
	if target != "/fallback" {
		t.Fatalf("expected /fallback, got %q", target)
	}
}

func TestManager_NilGuards(t *testing.T) {
	m := auth.New(time.Hour)
	defer m.Close()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/login", nil)
	user := &auth.User{ID: 1, Email: "user@example.com"}

	// Nil ResponseWriter
	if _, err := m.Login(nil, req, user); err == nil {
		t.Fatal("expected error on nil ResponseWriter")
	}

	// Nil Request
	if _, err := m.Login(rec, nil, user); err == nil {
		t.Fatal("expected error on nil Request")
	}

	// Nil User
	if _, err := m.Login(rec, req, nil); err == nil {
		t.Fatal("expected error on nil User")
	}

	// Logout with nil ResponseWriter should not panic
	m.Logout(nil, req)
	m.Logout(rec, nil)
}

func TestRateLimiter_Pruning(t *testing.T) {
	limiter := auth.NewRateLimiter(5, 50*time.Millisecond)
	limiter.Hit("10.0.0.1")
	limiter.Hit("10.0.0.2")

	if limiter.Attempts("10.0.0.1") != 1 {
		t.Fatal("expected 1 attempt")
	}

	// Wait for window to expire
	time.Sleep(60 * time.Millisecond)

	limiter.Prune()
	if limiter.Attempts("10.0.0.1") != 0 {
		t.Fatal("expected 0 attempts after expiration and prune")
	}
}

func TestRequired_APIContentNegotiation(t *testing.T) {
	mw := auth.Required("/login")
	handler := mw(func(ctx *router.Context) error {
		return ctx.Text("ok")
	})

	// JSON request
	req := httptest.NewRequest("GET", "/api/settings", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	ctx := router.NewTestContext(rec, req)

	_ = handler(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for API request, got %d", rec.Code)
	}

	// Standard browser request
	reqBrowser := httptest.NewRequest("GET", "/settings", nil)
	recBrowser := httptest.NewRecorder()
	ctxBrowser := router.NewTestContext(recBrowser, reqBrowser)

	_ = handler(ctxBrowser)
	if recBrowser.Code != http.StatusFound {
		t.Fatalf("expected 302 for browser request, got %d", recBrowser.Code)
	}
	if loc := recBrowser.Header().Get("Location"); loc != "/login" {
		t.Fatalf("expected redirect to /login, got %q", loc)
	}
}
