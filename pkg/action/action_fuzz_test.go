package action_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/misbakhul29/goks/pkg/action"
)

// FuzzActionOriginValidation verifies that arbitrary Origin and Host header strings
// never crash the server action security check and cannot bypass origin comparison.
func FuzzActionOriginValidation(f *testing.F) {
	seeds := []struct {
		origin string
		host   string
	}{
		{"http://localhost:3000", "localhost:3000"},
		{"https://example.com", "example.com"},
		{"http://attacker.com", "target.com"},
		{"http://target.com.attacker.com", "target.com"},
		{"http://target.com:8080", "target.com:3000"},
		{"null", "localhost:3000"},
		{"", "localhost:3000"},
		{"javascript:alert(1)", "localhost"},
		{"//evil.com", "localhost"},
	}

	for _, s := range seeds {
		f.Add(s.origin, s.host)
	}

	f.Fuzz(func(t *testing.T, origin string, host string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Action handler panicked on origin %q host %q: %v", origin, host, r)
			}
		}()

		req := httptest.NewRequest("POST", "/_goks/action", strings.NewReader(`{"name":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		if host != "" {
			req.Host = host
		}

		rec := httptest.NewRecorder()
		action.Handler().ServeHTTP(rec, req)
	})
}
