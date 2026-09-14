package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/misbakhul29/goks/pkg/auth"
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
