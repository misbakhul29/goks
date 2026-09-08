package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/misbakhul29/goks/pkg/router"
)

// DefaultJWT is the global JWT manager.
// Initialize it in your server config:
//
//	auth.DefaultJWT = auth.NewJWT("your-secret", 24*time.Hour)
var DefaultJWT *JWT

// JWTMiddleware validates the Bearer JWT token from the Authorization header.
// If valid, it injects the claims into the request context.
// If invalid/missing, it returns 401 Unauthorized.
func JWTMiddleware() router.MiddlewareFunc {
	return func(next router.Handler) router.Handler {
		return func(ctx *router.Context) error {
			j := DefaultJWT
			if j == nil {
				return fmt.Errorf("goks/auth: DefaultJWT not initialized")
			}

			authHeader := ctx.Header("Authorization")
			if authHeader == "" {
				ctx.Status(http.StatusUnauthorized).JSON(map[string]string{"error": "Authorization header required"})
				return nil
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := j.Verify(token)
			if err != nil {
				ctx.Status(http.StatusUnauthorized).JSON(map[string]string{"error": err.Error()})
				return nil
			}

			// Store claims in request context
			req := ctx.Request().WithContext(
				context.WithValue(ctx.Request().Context(), jwtClaimsKey, claims),
			)
			ctx.SetRequest(req)

			return next(ctx)
		}
	}
}

type jwtContextKey string

const jwtClaimsKey jwtContextKey = "goks_jwt_claims"

// GetClaims retrieves the JWT claims from the request context.
// Returns nil if no JWT middleware was applied or the token was invalid.
func GetClaims(r *http.Request) *JWTClaims {
	claims, _ := r.Context().Value(jwtClaimsKey).(*JWTClaims)
	return claims
}

// RequireRole is middleware that requires the authenticated user to have one of the given roles.
// Must be used after JWTMiddleware.
func RequireRole(roles ...string) router.MiddlewareFunc {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next router.Handler) router.Handler {
		return func(ctx *router.Context) error {
			claims := GetClaims(ctx.Request())
			if claims == nil {
				ctx.Status(http.StatusUnauthorized).JSON(map[string]string{"error": "Unauthorized"})
				return nil
			}
			for _, role := range claims.Roles {
				if allowed[role] {
					return next(ctx)
				}
			}
			ctx.Status(http.StatusForbidden).JSON(map[string]string{"error": "Forbidden"})
			return nil
		}
	}
}
