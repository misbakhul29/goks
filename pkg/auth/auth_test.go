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
