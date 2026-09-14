package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestOAuth_GenerateStateAndVerify(t *testing.T) {
	state, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState failed: %v", err)
	}
	if len(state) < 32 {
		t.Errorf("Expected state length >= 32, got %d", len(state))
	}

	// VerifyState success
	if !VerifyState(state, state) {
		t.Errorf("Expected VerifyState to succeed for identical states")
	}

	// VerifyState tampered state
	if VerifyState(state, state+"tampered") {
		t.Errorf("Expected VerifyState to fail for tampered state")
	}

	// Empty states
	if VerifyState("", state) || VerifyState(state, "") {
		t.Errorf("Expected VerifyState to fail for empty string")
	}
}

func TestOAuth_GeneratePKCE(t *testing.T) {
	verifier, challenge, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE failed: %v", err)
	}
	if verifier == "" || challenge == "" {
		t.Fatalf("Verifier or challenge is empty")
	}

	// Verify challenge matches SHA256 of verifier
	hash := sha256.Sum256([]byte(verifier))
	expectedChallenge := base64.RawURLEncoding.EncodeToString(hash[:])
	if challenge != expectedChallenge {
		t.Errorf("Expected challenge %s, got %s", expectedChallenge, challenge)
	}
}

func TestOAuth_Google_AuthURL(t *testing.T) {
	cfg := Config{
		ClientID:    "google-client-id-123",
		RedirectURL: "http://localhost:3000/api/auth/callback/google",
		Scopes:      []string{"openid", "email", "profile"},
	}
	g := Google(cfg)

	authURL := g.AuthURL(AuthOptions{
		State:         "random-state-456",
		CodeChallenge: "challenge-xyz",
		Prompt:        "consent",
		AccessType:    "offline",
	})

	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("Failed to parse AuthURL: %v", err)
	}

	if u.Host != "accounts.google.com" {
		t.Errorf("Expected accounts.google.com, got %s", u.Host)
	}

	q := u.Query()
	if q.Get("client_id") != "google-client-id-123" {
		t.Errorf("Expected client_id, got %s", q.Get("client_id"))
	}
	if q.Get("redirect_uri") != "http://localhost:3000/api/auth/callback/google" {
		t.Errorf("Expected redirect_uri, got %s", q.Get("redirect_uri"))
	}
	if q.Get("state") != "random-state-456" {
		t.Errorf("Expected state, got %s", q.Get("state"))
	}
	if q.Get("code_challenge") != "challenge-xyz" || q.Get("code_challenge_method") != "S256" {
		t.Errorf("Expected S256 code challenge, got %s (%s)", q.Get("code_challenge"), q.Get("code_challenge_method"))
	}
	if q.Get("access_type") != "offline" {
		t.Errorf("Expected access_type=offline")
	}
}

func TestOAuth_GitHub_AuthURL(t *testing.T) {
	cfg := Config{
		ClientID:    "github-client-id-789",
		RedirectURL: "http://localhost:3000/api/auth/callback/github",
	}
	gh := GitHub(cfg)

	authURL := gh.AuthURL(AuthOptions{
		State: "gh-state",
	})

	if !strings.Contains(authURL, "github.com/login/oauth/authorize") {
		t.Errorf("Expected GitHub authorize URL, got %s", authURL)
	}
	if !strings.Contains(authURL, "client_id=github-client-id-789") {
		t.Errorf("Expected client_id in URL, got %s", authURL)
	}
}

func TestOAuth_Exchange_And_UserInfo_Mock(t *testing.T) {
	// Mock OAuth Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			_ = r.ParseForm()
			code := r.Form.Get("code")
			if code != "valid_code" {
				http.Error(w, "invalid authorization code", http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token":  "mock_access_token_12345",
				"token_type":    "Bearer",
				"expires_in":    3600,
				"refresh_token": "mock_refresh_token_67890",
			})

		case "/oauth/userinfo":
			auth := r.Header.Get("Authorization")
			if auth != "Bearer mock_access_token_12345" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sub":     "google-user-999",
				"email":   "user@example.com",
				"name":    "GoKS Developer",
				"picture": "https://avatar.example.com/pic.png",
			})

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider := NewProvider("google", Endpoints{
		AuthURL:     server.URL + "/oauth/auth",
		TokenURL:    server.URL + "/oauth/token",
		UserInfoURL: server.URL + "/oauth/userinfo",
	}, Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost:3000/callback",
	})

	ctx := context.Background()

	// 1. Test successful token exchange
	token, err := provider.Exchange(ctx, "valid_code")
	if err != nil {
		t.Fatalf("Exchange failed: %v", err)
	}
	if token.AccessToken != "mock_access_token_12345" {
		t.Errorf("Expected access token mock_access_token_12345, got %s", token.AccessToken)
	}
	if token.ExpiresIn != 3600 {
		t.Errorf("Expected expires_in 3600, got %d", token.ExpiresIn)
	}

	// 2. Test fetching UserInfo
	userInfo, err := provider.UserInfo(ctx, token)
	if err != nil {
		t.Fatalf("UserInfo failed: %v", err)
	}
	if userInfo.ID != "google-user-999" {
		t.Errorf("Expected user ID google-user-999, got %s", userInfo.ID)
	}
	if userInfo.Email != "user@example.com" {
		t.Errorf("Expected email user@example.com, got %s", userInfo.Email)
	}
	if userInfo.Name != "GoKS Developer" {
		t.Errorf("Expected name GoKS Developer, got %s", userInfo.Name)
	}
	if userInfo.AvatarURL != "https://avatar.example.com/pic.png" {
		t.Errorf("Expected avatar URL, got %s", userInfo.AvatarURL)
	}

	// 3. Test invalid code error handling
	_, err = provider.Exchange(ctx, "invalid_code")
	if err == nil {
		t.Errorf("Expected error for invalid code")
	}
}
