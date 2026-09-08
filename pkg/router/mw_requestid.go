package router

import (
	"crypto/rand"
	"encoding/hex"
)

// generateID creates a simple random 16-byte hex string.
func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// RequestID is a middleware that injects a unique X-Request-Id header to each request
// if one is not already present, and makes it available in the response headers.
func RequestID() MiddlewareFunc {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			reqID := ctx.Request().Header.Get("X-Request-Id")
			if reqID == "" {
				reqID = generateID()
				ctx.Request().Header.Set("X-Request-Id", reqID)
			}
			ctx.SetHeader("X-Request-Id", reqID)
			
			return next(ctx)
		}
	}
}
