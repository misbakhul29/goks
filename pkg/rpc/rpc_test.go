package rpc_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/misbakhul29/goks/pkg/rpc"
)

type GreetInput struct {
	Name string `json:"name"`
}

type GreetOutput struct {
	Message string `json:"message"`
}

func greet(in GreetInput) (GreetOutput, error) {
	if in.Name == "" {
		return GreetOutput{}, errors.New("name is required")
	}
	if in.Name == "panic" {
		panic("forced panic inside rpc handler")
	}
	return GreetOutput{Message: "Hello " + in.Name}, nil
}

func init() {
	rpc.Register("greet", greet)
}

func TestRPC_Success(t *testing.T) {
	handler := rpc.Handler()

	body, _ := json.Marshal(GreetInput{Name: "Gopher"})
	req := httptest.NewRequest(http.MethodPost, "/__goks_rpc?method=greet", bytes.NewReader(body))
	req.Host = "localhost:3000"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var out GreetOutput
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if out.Message != "Hello Gopher" {
		t.Fatalf("unexpected message: %s", out.Message)
	}
}

func TestRPC_CrossOrigin_Forbidden(t *testing.T) {
	handler := rpc.Handler()

	tests := []struct {
		name         string
		origin       string
		secFetchSite string
		want         int
	}{
		{name: "different domain", origin: "http://evil.com", want: http.StatusForbidden},
		{name: "suffix bypass attempt", origin: "http://myapp.com.evil.com", want: http.StatusForbidden},
		{name: "prefix bypass attempt", origin: "http://evil-myapp.com", want: http.StatusForbidden},
		{name: "cross-site Sec-Fetch-Site", secFetchSite: "cross-site", want: http.StatusForbidden},
		{name: "same origin valid", origin: "http://myapp.com", want: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(GreetInput{Name: "Tester"})
			req := httptest.NewRequest(http.MethodPost, "http://myapp.com/__goks_rpc?method=greet", bytes.NewReader(body))
			req.Host = "myapp.com"
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.secFetchSite != "" {
				req.Header.Set("Sec-Fetch-Site", tt.secFetchSite)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.want {
				t.Fatalf("expected status %d, got %d: %s", tt.want, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestRPC_PanicRecovery(t *testing.T) {
	handler := rpc.Handler()

	body, _ := json.Marshal(GreetInput{Name: "panic"})
	req := httptest.NewRequest(http.MethodPost, "/__goks_rpc?method=greet", bytes.NewReader(body))
	req.Host = "localhost:3000"
	rec := httptest.NewRecorder()

	// Should not panic the test process
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on panic recovery, got %d", rec.Code)
	}
}

func TestRPC_MethodNotFound(t *testing.T) {
	handler := rpc.Handler()

	req := httptest.NewRequest(http.MethodPost, "/__goks_rpc?method=unknown_function", bytes.NewReader([]byte("{}")))
	req.Host = "localhost:3000"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown RPC method, got %d", rec.Code)
	}
}
