package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// JWTClaims represents the payload of a JWT token.
type JWTClaims struct {
	Sub   uint     `json:"sub"`   // User ID
	Email string   `json:"email"` // User email
	Name  string   `json:"name"`  // User display name
	Roles []string `json:"roles"` // User roles
	Exp   int64    `json:"exp"`   // Unix expiry
	Iat   int64    `json:"iat"`   // Issued-at
}

// IsExpired returns true if the token has expired.
func (c *JWTClaims) IsExpired() bool {
	return time.Now().Unix() > c.Exp
}

// ToUser converts claims back into a User struct.
func (c *JWTClaims) ToUser() *User {
	return &User{
		ID:    c.Sub,
		Email: c.Email,
		Name:  c.Name,
		Roles: c.Roles,
	}
}

// JWT provides simple HS256 JWT sign/verify without external dependencies.
type JWT struct {
	secret []byte
	ttl    time.Duration
}

// NewJWT creates a new JWT manager.
// secret: signing key (keep this secret!)
// ttl: token lifetime (e.g., 24*time.Hour)
func NewJWT(secret string, ttl time.Duration) *JWT {
	return &JWT{secret: []byte(secret), ttl: ttl}
}

// Sign creates a signed JWT token for the given user.
func (j *JWT) Sign(user *User) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		Sub:   user.ID,
		Email: user.Email,
		Name:  user.Name,
		Roles: user.Roles,
		Iat:   now.Unix(),
		Exp:   now.Add(j.ttl).Unix(),
	}

	header := base64Encode([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("goks/auth: jwt sign: %w", err)
	}
	encodedPayload := base64Encode(payload)
	sig := j.sign(header + "." + encodedPayload)

	return header + "." + encodedPayload + "." + sig, nil
}

// Verify parses and validates a JWT token string.
// Returns the claims if valid.
func (j *JWT) Verify(token string) (*JWTClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("goks/auth: jwt: malformed token")
	}

	// Verify signature
	expected := j.sign(parts[0] + "." + parts[1])
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return nil, fmt.Errorf("goks/auth: jwt: invalid signature")
	}

	// Decode payload
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("goks/auth: jwt: invalid payload encoding")
	}

	var claims JWTClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("goks/auth: jwt: invalid payload: %w", err)
	}

	if claims.IsExpired() {
		return nil, fmt.Errorf("goks/auth: jwt: token expired")
	}

	return &claims, nil
}

func (j *JWT) sign(data string) string {
	mac := hmac.New(sha256.New, j.secret)
	mac.Write([]byte(data))
	return base64Encode(mac.Sum(nil))
}

func base64Encode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}
