// Package oauth provides zero-dependency OAuth2 social authentication for GoKS.
// It includes built-in providers for Google and GitHub, PKCE (RFC 7636) code verification,
// timing-attack resistant state validation, and unified user profile mapping.
package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Endpoints contains the OAuth2 provider service URLs.
type Endpoints struct {
	AuthURL     string
	TokenURL    string
	UserInfoURL string
}

// Config configures an OAuth2 provider.
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// Token represents the OAuth2 token received from the authorization server.
type Token struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresIn    int64     `json:"expires_in,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
	IDToken      string    `json:"id_token,omitempty"`
	Scope        string    `json:"scope,omitempty"`
}

// UserInfo represents a normalized user profile from any OAuth2 provider.
type UserInfo struct {
	ID        string         `json:"id"`
	Email     string         `json:"email"`
	Name      string         `json:"name"`
	AvatarURL string         `json:"avatar_url"`
	Provider  string         `json:"provider"`
	Raw       map[string]any `json:"raw"`
}

// Provider represents a configured OAuth2 identity provider.
type Provider struct {
	name       string
	endpoints  Endpoints
	cfg        Config
	httpClient *http.Client
}

// NewProvider creates a custom OAuth2 provider.
func NewProvider(name string, endpoints Endpoints, cfg Config) *Provider {
	return &Provider{
		name:      name,
		endpoints: endpoints,
		cfg:       cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Google creates a configured Google OAuth2 provider.
func Google(cfg Config) *Provider {
	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{"openid", "email", "profile"}
	}
	return NewProvider("google", Endpoints{
		AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:    "https://oauth2.googleapis.com/token",
		UserInfoURL: "https://www.googleapis.com/oauth2/v3/userinfo",
	}, cfg)
}

// GitHub creates a configured GitHub OAuth2 provider.
func GitHub(cfg Config) *Provider {
	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{"read:user", "user:email"}
	}
	return NewProvider("github", Endpoints{
		AuthURL:     "https://github.com/login/oauth/authorize",
		TokenURL:    "https://github.com/login/oauth/access_token",
		UserInfoURL: "https://api.github.com/user",
	}, cfg)
}

// AuthOptions configures options for generating the authorization URL.
type AuthOptions struct {
	State         string
	CodeChallenge string
	Prompt        string // e.g. "consent" or "select_account"
	AccessType    string // e.g. "offline" for refresh tokens
}

// AuthURL generates the redirect URL for the user to authenticate with the provider.
func (p *Provider) AuthURL(opts AuthOptions) string {
	vals := url.Values{
		"response_type": {"code"},
		"client_id":     {p.cfg.ClientID},
		"redirect_uri":  {p.cfg.RedirectURL},
	}

	if len(p.cfg.Scopes) > 0 {
		vals.Set("scope", strings.Join(p.cfg.Scopes, " "))
	}
	if opts.State != "" {
		vals.Set("state", opts.State)
	}
	if opts.CodeChallenge != "" {
		vals.Set("code_challenge", opts.CodeChallenge)
		vals.Set("code_challenge_method", "S256")
	}
	if opts.Prompt != "" {
		vals.Set("prompt", opts.Prompt)
	}
	if opts.AccessType != "" {
		vals.Set("access_type", opts.AccessType)
	}

	sep := "?"
	if strings.Contains(p.endpoints.AuthURL, "?") {
		sep = "&"
	}
	return p.endpoints.AuthURL + sep + vals.Encode()
}

// ExchangeOptions configures the authorization code exchange.
type ExchangeOptions struct {
	CodeVerifier string
}

// Exchange exchanges an authorization code for an access token.
func (p *Provider) Exchange(ctx context.Context, code string, opts ...ExchangeOptions) (*Token, error) {
	if code == "" {
		return nil, errors.New("oauth: authorization code cannot be empty")
	}

	vals := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {p.cfg.RedirectURL},
		"client_id":     {p.cfg.ClientID},
		"client_secret": {p.cfg.ClientSecret},
	}

	if len(opts) > 0 && opts[0].CodeVerifier != "" {
		vals.Set("code_verifier", opts[0].CodeVerifier)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoints.TokenURL, strings.NewReader(vals.Encode()))
	if err != nil {
		return nil, fmt.Errorf("oauth: failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oauth: token exchange network error: %w", err)
	}
	defer resp.Body.Close()

	// Limit response to 1MB to prevent decompression or memory exhaustion
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("oauth: failed to read token response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("oauth: token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tok Token
	// GitHub might return application/x-www-form-urlencoded if Accept header was ignored
	if strings.Contains(resp.Header.Get("Content-Type"), "application/json") || strings.HasPrefix(strings.TrimSpace(string(body)), "{") {
		if err := json.Unmarshal(body, &tok); err != nil {
			return nil, fmt.Errorf("oauth: failed to parse token JSON: %w", err)
		}
	} else {
		parsed, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, fmt.Errorf("oauth: failed to parse form-encoded token: %w", err)
		}
		tok.AccessToken = parsed.Get("access_token")
		tok.TokenType = parsed.Get("token_type")
		tok.RefreshToken = parsed.Get("refresh_token")
		tok.Scope = parsed.Get("scope")
		if exp := parsed.Get("expires_in"); exp != "" {
			if sec, err := strconv.ParseInt(exp, 10, 64); err == nil {
				tok.ExpiresIn = sec
			}
		}
	}

	if tok.AccessToken == "" {
		return nil, fmt.Errorf("oauth: no access_token in response: %s", string(body))
	}

	if tok.ExpiresIn > 0 {
		tok.Expiry = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	}

	return &tok, nil
}

// UserInfo fetches the user's profile from the provider's UserInfo endpoint.
func (p *Provider) UserInfo(ctx context.Context, token *Token) (*UserInfo, error) {
	if token == nil || token.AccessToken == "" {
		return nil, errors.New("oauth: invalid or empty token")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.endpoints.UserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("oauth: failed to create userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/json")
	// GitHub recommends User-Agent
	req.Header.Set("User-Agent", "GoKS-OAuth2/v0.15.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oauth: userinfo request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("oauth: failed to read userinfo response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("oauth: userinfo request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("oauth: failed to parse userinfo JSON: %w", err)
	}

	info := &UserInfo{
		Provider: p.name,
		Raw:      raw,
	}

	switch p.name {
	case "google":
		if sub, ok := raw["sub"].(string); ok {
			info.ID = sub
		}
		if email, ok := raw["email"].(string); ok {
			info.Email = email
		}
		if name, ok := raw["name"].(string); ok {
			info.Name = name
		}
		if pic, ok := raw["picture"].(string); ok {
			info.AvatarURL = pic
		}
	case "github":
		if id, ok := raw["id"].(float64); ok {
			info.ID = strconv.FormatInt(int64(id), 10)
		}
		if email, ok := raw["email"].(string); ok && email != "" {
			info.Email = email
		}
		if name, ok := raw["name"].(string); ok && name != "" {
			info.Name = name
		} else if login, ok := raw["login"].(string); ok {
			info.Name = login
		}
		if avatar, ok := raw["avatar_url"].(string); ok {
			info.AvatarURL = avatar
		}

		// If GitHub user has private email, attempt to fetch from /user/emails
		if info.Email == "" {
			info.Email = p.fetchGitHubPrimaryEmail(ctx, token.AccessToken)
		}
	default:
		// Generic mapping
		for _, k := range []string{"id", "sub", "user_id"} {
			if val, ok := raw[k]; ok {
				info.ID = fmt.Sprintf("%v", val)
				break
			}
		}
		for _, k := range []string{"email", "mail"} {
			if val, ok := raw[k].(string); ok {
				info.Email = val
				break
			}
		}
		for _, k := range []string{"name", "display_name", "username"} {
			if val, ok := raw[k].(string); ok {
				info.Name = val
				break
			}
		}
		for _, k := range []string{"avatar_url", "picture", "avatar"} {
			if val, ok := raw[k].(string); ok {
				info.AvatarURL = val
				break
			}
		}
	}

	return info, nil
}

func (p *Provider) fetchGitHubPrimaryEmail(ctx context.Context, token string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "GoKS-OAuth2/v0.15.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<18)).Decode(&emails); err != nil {
		return ""
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email
		}
	}
	if len(emails) > 0 {
		return emails[0].Email
	}
	return ""
}

// GenerateState produces a cryptographically secure, URL-safe random state token.
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("oauth: failed to read crypto random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GeneratePKCE creates a code verifier and its corresponding SHA256 code challenge (RFC 7636).
func GeneratePKCE() (verifier, challenge string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("oauth: failed to generate PKCE verifier: %w", err)
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)
	hash := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(hash[:])
	return verifier, challenge, nil
}

// VerifyState verifies that receivedState matches expectedState in constant time
// to prevent timing attacks against CSRF protection.
func VerifyState(expectedState, receivedState string) bool {
	if expectedState == "" || receivedState == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expectedState), []byte(receivedState)) == 1
}
