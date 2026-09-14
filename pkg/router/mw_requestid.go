package router

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

// generateID creates a simple random 16-byte hex string.
func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

const requestIDKey contextKey = "goks_request_id"

// ContextWithRequestID injects a request ID into a context.Context.
func ContextWithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFromContext retrieves the request ID from a context.Context if present.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
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

			req := ctx.Request().WithContext(ContextWithRequestID(ctx.Request().Context(), reqID))
			ctx.SetRequest(req)

			return next(ctx)
		}
	}
}
