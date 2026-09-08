package router

import (
	"crypto/subtle"
	"net/http"
)

// BasicAuth is a middleware that requires HTTP Basic Authentication.
// 'users' is a map where keys are usernames and values are passwords.
func BasicAuth(users map[string]string) MiddlewareFunc {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			user, pass, ok := ctx.Request().BasicAuth()
			if !ok {
				ctx.SetHeader("WWW-Authenticate", `Basic realm="Restricted"`)
				ctx.Status(http.StatusUnauthorized).Text("Unauthorized")
				return nil
			}

			expectedPass, exists := users[user]
			if !exists || subtle.ConstantTimeCompare([]byte(pass), []byte(expectedPass)) != 1 {
				ctx.SetHeader("WWW-Authenticate", `Basic realm="Restricted"`)
				ctx.Status(http.StatusUnauthorized).Text("Unauthorized")
				return nil
			}

			// Store the authenticated username in the request context/headers for later use if needed
			// ctx.Set("user", user) -> assuming ctx has a generic Set method, otherwise we can skip

			return next(ctx)
		}
	}
}
