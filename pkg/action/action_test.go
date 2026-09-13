package action_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/misbakhul29/goks/pkg/action"
)

func init() {
	action.Register("createItem", func(ctx *action.Context) (any, error) {
		name := ctx.Get("name")
		if name == "" {
			return nil, errors.New("name is required")
		}
		return map[string]string{"id": "item_123", "name": name}, nil
	})
}

func TestAction_JSONMode(t *testing.T) {
	handler := action.Handler()

	form := url.Values{}
	form.Set("name", "GoKS Action")

	req := httptest.NewRequest(http.MethodPost, "/__goks_action?name=createItem", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-GoKS-Action", "1")
	req.Host = "localhost:3000"

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp action.Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Fatalf("expected success to be true")
	}
}

func TestAction_HTMLFormRedirect(t *testing.T) {
	handler := action.Handler()

	form := url.Values{}
	form.Set("name", "Standard Form Post")

	req := httptest.NewRequest(http.MethodPost, "/__goks_action?name=createItem", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "/dashboard")
	req.Host = "localhost:3000"

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 See Other redirect, got %d", rec.Code)
	}

	loc := rec.Header().Get("Location")
	if loc != "/dashboard" {
		t.Fatalf("expected redirect to /dashboard, got %q", loc)
	}
}

func TestAction_CrossOrigin_Rejected(t *testing.T) {
	handler := action.Handler()

	form := url.Values{}
	form.Set("name", "Malicious")

	req := httptest.NewRequest(http.MethodPost, "/__goks_action?name=createItem", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://evil-site.com")
	req.Host = "localhost:3000"

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for cross-origin action, got %d", rec.Code)
	}
}

func TestAction_NotFound(t *testing.T) {
	handler := action.Handler()

	req := httptest.NewRequest(http.MethodPost, "/__goks_action?name=nonExistent", nil)
	req.Host = "localhost:3000"

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown action, got %d", rec.Code)
	}
}

func TestAction_URLHelper(t *testing.T) {
	u := action.URL("deleteUser")
	if u != "/__goks_action?name=deleteUser" {
		t.Fatalf("expected /__goks_action?name=deleteUser, got %q", u)
	}
}
