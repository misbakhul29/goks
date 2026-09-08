// Package auth provides authentication and session management for GoKS.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/misbakhulmunir/goks/pkg/router"
)

// User represents an authenticated user.
type User struct {
	ID    uint
	Email string
	Name  string
	Roles []string
}

// Session holds auth data for a browser session.
type Session struct {
	Token     string
	User      *User
	ExpiresAt time.Time
}

// Manager handles session storage and authentication.
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	ttl      time.Duration
}

// Default is the global auth manager.
var Default = New(24 * time.Hour)

// New creates a new auth Manager with the given session TTL.
func New(ttl time.Duration) *Manager {
	m := &Manager{
		sessions: make(map[string]*Session),
		ttl:      ttl,
	}
	go m.gcLoop()
	return m
}

// Login creates a new session for the given user and sets a cookie.
func (m *Manager) Login(w http.ResponseWriter, user *User) (*Session, error) {
	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	sess := &Session{
		Token:     token,
		User:      user,
		ExpiresAt: time.Now().Add(m.ttl),
	}

	m.mu.Lock()
	m.sessions[token] = sess
	m.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     "goks_session",
		Value:    token,
		Path:     "/",
		Expires:  sess.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return sess, nil
}

// Logout invalidates the session and clears the cookie.
func (m *Manager) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("goks_session")
	if err == nil {
		m.mu.Lock()
		delete(m.sessions, cookie.Value)
		m.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{
		Name:    "goks_session",
		Value:   "",
		Path:    "/",
		Expires: time.Unix(0, 0),
		MaxAge:  -1,
	})
}

// SessionFromRequest retrieves the session from the request cookie.
func (m *Manager) SessionFromRequest(r *http.Request) (*Session, bool) {
	cookie, err := r.Cookie("goks_session")
	if err != nil {
		return nil, false
	}
	m.mu.RLock()
	sess, ok := m.sessions[cookie.Value]
	m.mu.RUnlock()
	if !ok || time.Now().After(sess.ExpiresAt) {
		return nil, false
	}
	return sess, true
}

// -----------------------------------------------------------------------
// Middleware
// -----------------------------------------------------------------------

type contextKey string

const sessionKey contextKey = "goks_session"

// Required is middleware that requires an authenticated session.
// Redirects to /login if not authenticated.
func Required() router.MiddlewareFunc {
	return func(next router.Handler) router.Handler {
		return func(ctx *router.Context) error {
			sess, ok := Default.SessionFromRequest(ctx.Request())
			if !ok {
				return ctx.Redirect("/login", http.StatusFound)
			}
			// Store user in request context
			req := ctx.Request().WithContext(
				context.WithValue(ctx.Request().Context(), sessionKey, sess),
			)
			*ctx.Request() = *req
			return next(ctx)
		}
	}
}

// CurrentUser retrieves the authenticated user from the request context.
// Returns nil if not authenticated.
func CurrentUser(r *http.Request) *User {
	sess, ok := r.Context().Value(sessionKey).(*Session)
	if !ok || sess == nil {
		return nil
	}
	return sess.User
}

// -----------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (m *Manager) gcLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		m.mu.Lock()
		for k, s := range m.sessions {
			if now.After(s.ExpiresAt) {
				delete(m.sessions, k)
			}
		}
		m.mu.Unlock()
	}
}
