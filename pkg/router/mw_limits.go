package router

import (
	"context"
	"net/http"
	"time"
)

// MaxBytes limits the size of the request body to prevent abuse.
func MaxBytes(maxSize int64) MiddlewareFunc {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			if ctx.Request().ContentLength > maxSize {
				ctx.Status(http.StatusRequestEntityTooLarge).Text("Request Entity Too Large")
				return nil
			}
			
			// Protect against streams without Content-Length
			ctx.Request().Body = http.MaxBytesReader(ctx.Response(), ctx.Request().Body, maxSize)
			
			return next(ctx)
		}
	}
}

// Timeout adds a timeout to the request context.
func Timeout(d time.Duration) MiddlewareFunc {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			reqCtx, cancel := context.WithTimeout(ctx.Request().Context(), d)
			defer cancel()
			
			// Replace the request with the new context
			req := ctx.Request().WithContext(reqCtx)
			ctx.r = req
			
			// Use a channel to run the handler and check for timeout
			done := make(chan error, 1)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						// Pass panic up to the Recover middleware if any
						panic(r)
					}
				}()
				done <- next(ctx)
			}()
			
			select {
			case <-reqCtx.Done():
				// Context timeout exceeded
				ctx.Status(http.StatusGatewayTimeout).Text("Gateway Timeout")
				return nil
			case err := <-done:
				return err
			}
		}
	}
}
