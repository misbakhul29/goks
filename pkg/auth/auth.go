// Package auth provides authentication and session management for GoKS.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/misbakhul29/goks/pkg/router"
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

// LoginOption configures session creation behavior.
type LoginOption func(*loginConfig)

type loginConfig struct {
	ttl        time.Duration
	hasTTL     bool
	rememberMe bool
}

// WithTTL sets a custom TTL duration for the session.
func WithTTL(ttl time.Duration) LoginOption {
	return func(c *loginConfig) {
		c.ttl = ttl
		c.hasTTL = true
	}
}

// WithRememberMe specifies if the session should persist across browser restarts (default: 30 days).
func WithRememberMe(remember bool) LoginOption {
	return func(c *loginConfig) {
		c.rememberMe = remember
	}
}

// Manager handles session storage and authentication.
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	ttl      time.Duration
	stopGC   chan struct{}
}

// Default is the global auth manager.
var Default = New(24 * time.Hour)

// New creates a new auth Manager with the given session TTL.
func New(ttl time.Duration) *Manager {
	m := &Manager{
		sessions: make(map[string]*Session),
		ttl:      ttl,
		stopGC:   make(chan struct{}),
	}
	go m.gcLoop()
	return m
}

// Close stops the background session garbage collector.
func (m *Manager) Close() {
	if m.stopGC != nil {
		select {
		case <-m.stopGC:
			// already closed
		default:
			close(m.stopGC)
		}
	}
}

// Login creates a new session for the given user and sets a cookie.
// It invalidates any existing session token from the request to prevent session fixation.
func (m *Manager) Login(w http.ResponseWriter, r *http.Request, user *User, opts ...LoginOption) (*Session, error) {
	if w == nil || r == nil {
		return nil, errors.New("auth: response writer and request cannot be nil")
	}
	if user == nil {
		return nil, errors.New("auth: cannot create session for nil user")
	}

	cfg := loginConfig{ttl: m.ttl}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.rememberMe && !cfg.hasTTL {
		cfg.ttl = 30 * 24 * time.Hour
	}

	if oldCookie, err := r.Cookie("goks_session"); err == nil && oldCookie.Value != "" {
		m.mu.Lock()
		delete(m.sessions, oldCookie.Value)
		m.mu.Unlock()
	}

	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	sess := &Session{
		Token:     token,
		User:      user,
		ExpiresAt: time.Now().Add(cfg.ttl),
	}

	m.mu.Lock()
	m.sessions[token] = sess
	m.mu.Unlock()

	// Set Secure flag if the request is over HTTPS (direct TLS or behind a reverse proxy).
	secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"

	cookie := &http.Cookie{
		Name:     "goks_session",
		Value:    token,
		Path:     "/",
		Expires:  sess.ExpiresAt,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
	if cfg.rememberMe {
		cookie.MaxAge = int(cfg.ttl.Seconds())
	}

	http.SetCookie(w, cookie)
	reqCtx := r.Context()
	if reqCtx == nil {
		reqCtx = context.Background()
	}
	triggerLoginHooks(reqCtx, user, r)
	return sess, nil
}

// Logout invalidates the session and clears the cookie.
func (m *Manager) Logout(w http.ResponseWriter, r *http.Request) {
	if w == nil {
		return
	}
	var user *User
	if r != nil {
		if cookie, err := r.Cookie("goks_session"); err == nil {
			m.mu.Lock()
			if sess, ok := m.sessions[cookie.Value]; ok {
				user = sess.User
				delete(m.sessions, cookie.Value)
			}
			m.mu.Unlock()
		}
	}
	secure := r != nil && (r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https")
	http.SetCookie(w, &http.Cookie{
		Name:     "goks_session",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
	if user != nil {
		var reqCtx context.Context = context.Background()
		if r != nil && r.Context() != nil {
			reqCtx = r.Context()
		}
		triggerLogoutHooks(reqCtx, user, r)
	}
}

// SessionFromRequest retrieves the session from the request cookie.
func (m *Manager) SessionFromRequest(r *http.Request) (*Session, bool) {
	if r == nil {
		return nil, false
	}
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

// ContextWithSession injects the authenticated session into the context.
func ContextWithSession(ctx context.Context, sess *Session) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, sessionKey, sess)
}

// Required is middleware that requires an authenticated session.
// If unauthenticated, it records the current request URI as the intended destination and redirects to /login (or custom path).
func Required(redirectTo ...string) router.MiddlewareFunc {
	loginURL := "/login"
	if len(redirectTo) > 0 && redirectTo[0] != "" {
		loginURL = redirectTo[0]
	}

	return func(next router.Handler) router.Handler {
		return func(ctx *router.Context) error {
			sess, ok := Default.SessionFromRequest(ctx.Request())
			if !ok {
				// API/JSON content-negotiation: return 401 instead of redirecting
				if strings.Contains(ctx.Header("Accept"), "application/json") || strings.HasPrefix(ctx.Path(), "/api/") {
					return ctx.Status(http.StatusUnauthorized).JSON(map[string]string{
						"error": "authentication required",
					})
				}
				if ctx.Request() != nil && ctx.Request().Method == http.MethodGet && ctx.Request().URL != nil {
					SetIntendedURL(ctx.Response(), ctx.Request(), ctx.Request().URL.RequestURI())
				}
				return ctx.Redirect(loginURL, http.StatusFound)
			}
			req := ctx.Request().WithContext(
				ContextWithSession(ctx.Request().Context(), sess),
			)
			ctx.SetRequest(req)
			return next(ctx)
		}
	}
}

// Guest is middleware that ensures the user is NOT authenticated.
// If the user is authenticated, it redirects to redirectTo (defaults to "/dashboard").
func Guest(redirectTo ...string) router.MiddlewareFunc {
	dest := "/dashboard"
	if len(redirectTo) > 0 && redirectTo[0] != "" {
		dest = redirectTo[0]
	}

	return func(next router.Handler) router.Handler {
		return func(ctx *router.Context) error {
			if _, ok := Default.SessionFromRequest(ctx.Request()); ok {
				return ctx.Redirect(dest, http.StatusFound)
			}
			return next(ctx)
		}
	}
}

// CurrentUser retrieves the authenticated user from the request context or session cookie.
// Returns nil if not authenticated.
func CurrentUser(r *http.Request) *User {
	if r == nil {
		return nil
	}
	sess, ok := r.Context().Value(sessionKey).(*Session)
	if ok && sess != nil {
		return sess.User
	}
	if s, found := Default.SessionFromRequest(r); found && s != nil {
		return s.User
	}
	return nil
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
	for {
		select {
		case <-m.stopGC:
			return
		case <-ticker.C:
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
}
