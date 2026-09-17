package auth

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/misbakhul29/goks/pkg/router"
)

// RateLimiter tracks and limits execution frequency by key.
type RateLimiter struct {
	mu          sync.Mutex
	attempts    map[string][]time.Time
	maxAttempts int
	window      time.Duration
}

// NewRateLimiter creates a new thread-safe rate limiter.
func NewRateLimiter(maxAttempts int, window time.Duration) *RateLimiter {
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	if window <= 0 {
		window = time.Minute
	}
	return &RateLimiter{
		attempts:    make(map[string][]time.Time),
		maxAttempts: maxAttempts,
		window:      window,
	}
}

// Hit records an attempt for the given key and returns the total active attempts.
func (l *RateLimiter) Hit(key string) int {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	var valid []time.Time
	for _, t := range l.attempts[key] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	valid = append(valid, now)
	l.attempts[key] = valid
	return len(valid)
}

// TooManyAttempts checks if the key has exceeded the allowed attempt threshold.
func (l *RateLimiter) TooManyAttempts(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	count := 0
	for _, t := range l.attempts[key] {
		if t.After(cutoff) {
			count++
		}
	}
	return count >= l.maxAttempts
}

// Clear resets recorded attempts for the given key.
func (l *RateLimiter) Clear(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}

// AvailableIn returns the remaining duration before the oldest attempt expires.
func (l *RateLimiter) AvailableIn(key string) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()

	times, ok := l.attempts[key]
	if !ok || len(times) == 0 {
		return 0
	}

	oldest := times[0]
	expiry := oldest.Add(l.window)
	remaining := time.Until(expiry)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Attempts returns the count of active attempts within the current window.
func (l *RateLimiter) Attempts(key string) int {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := time.Now().Add(-l.window)
	count := 0
	for _, t := range l.attempts[key] {
		if t.After(cutoff) {
			count++
		}
	}
	return count
}

// DefaultLimiter is the global auth rate limiter (5 attempts per minute).
var DefaultLimiter = NewRateLimiter(5, time.Minute)

// Throttle returns a router middleware that limits request rates.
// Key resolution defaults to client IP address.
func Throttle(maxAttempts int, window time.Duration, keyFunc ...func(ctx *router.Context) string) router.MiddlewareFunc {
	limiter := NewRateLimiter(maxAttempts, window)

	return func(next router.Handler) router.Handler {
		return func(ctx *router.Context) error {
			key := ""
			if len(keyFunc) > 0 && keyFunc[0] != nil {
				key = keyFunc[0](ctx)
			} else {
				key = resolveClientIP(ctx.Request())
			}

			if limiter.TooManyAttempts(key) {
				sec := int(limiter.AvailableIn(key).Seconds())
				if sec <= 0 {
					sec = 1
				}
				ctx.SetHeader("Retry-After", fmt.Sprintf("%d", sec))

				accept := ctx.Header("Accept")
				if strings.Contains(accept, "application/json") {
					return ctx.Status(http.StatusTooManyRequests).JSON(map[string]any{
						"error":       "Too many requests. Please try again later.",
						"retry_after": sec,
					})
				}
				return ctx.Status(http.StatusTooManyRequests).Text(fmt.Sprintf("Too many requests. Please try again in %d seconds.", sec))
			}

			limiter.Hit(key)
			return next(ctx)
		}
	}
}

func resolveClientIP(r *http.Request) string {
	if r == nil {
		return "127.0.0.1"
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	idx := strings.LastIndex(r.RemoteAddr, ":")
	if idx != -1 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}
