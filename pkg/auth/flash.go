package auth

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	flashCookiePrefix = "goks_flash_"
	intendedURLCookie = "goks_intended_url"
)

// SetFlash stores a one-time flash message in a cookie for the given key.
func SetFlash(w http.ResponseWriter, r *http.Request, key, val string) {
	cookieName := flashCookiePrefix + sanitizeCookieName(key)
	encoded := base64.RawURLEncoding.EncodeToString([]byte(val))
	secure := r != nil && (r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https")

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    encoded,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// GetFlash retrieves and immediately clears the one-time flash message for the given key.
func GetFlash(w http.ResponseWriter, r *http.Request, key string) string {
	if r == nil {
		return ""
	}
	cookieName := flashCookiePrefix + sanitizeCookieName(key)
	cookie, err := r.Cookie(cookieName)
	if err != nil || cookie.Value == "" {
		return ""
	}

	// Invalidate the flash cookie on read
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
	})

	decoded, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return ""
	}
	return string(decoded)
}

// SetIntendedURL stores the requested URL before redirecting to login.
func SetIntendedURL(w http.ResponseWriter, r *http.Request, targetURL string) {
	if !isValidInternalURL(targetURL) {
		return
	}
	secure := r != nil && (r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https")
	encoded := base64.RawURLEncoding.EncodeToString([]byte(targetURL))

	http.SetCookie(w, &http.Cookie{
		Name:     intendedURLCookie,
		Value:    encoded,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// GetIntendedURL returns the intended URL from cookie, or fallbackURL if none or invalid.
// It clears the intended URL cookie once retrieved.
func GetIntendedURL(w http.ResponseWriter, r *http.Request, fallbackURL string) string {
	if fallbackURL == "" {
		fallbackURL = "/"
	}
	if r == nil {
		return fallbackURL
	}

	cookie, err := r.Cookie(intendedURLCookie)
	if err != nil || cookie.Value == "" {
		return fallbackURL
	}

	// Invalidate cookie
	http.SetCookie(w, &http.Cookie{
		Name:     intendedURLCookie,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
	})

	raw, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return fallbackURL
	}
	target := string(raw)
	if !isValidInternalURL(target) {
		return fallbackURL
	}
	return target
}

// isValidInternalURL validates that the target URL is a safe, relative path
// to prevent open redirect vulnerabilities.
func isValidInternalURL(target string) bool {
	if target == "" {
		return false
	}
	// Must start with '/' and not '//' (protocol-relative) or contain backslashes
	if !strings.HasPrefix(target, "/") || strings.HasPrefix(target, "//") || strings.Contains(target, `\`) {
		return false
	}
	u, err := url.Parse(target)
	if err != nil {
		return false
	}
	return u.Host == "" && u.Scheme == ""
}

func sanitizeCookieName(s string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, s)
}
