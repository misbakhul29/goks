// Package router - Common middleware for GoKS.
package router

import (
	"log"
	"net/http"
	"time"
)

// Logger is middleware that logs each request with method, path, status, and duration.
func Logger() MiddlewareFunc {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			start := time.Now()
			err := next(ctx)
			log.Printf("[GoKS] %s %s %v", ctx.Method(), ctx.Path(), time.Since(start))
			return err
		}
	}
}

// Recover is middleware that recovers from panics and returns a 500 error.
func Recover() MiddlewareFunc {
	return func(next Handler) Handler {
		return func(ctx *Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[GoKS] PANIC: %v", r)
					ctx.Status(http.StatusInternalServerError).Text("Internal Server Error")
				}
			}()
			return next(ctx)
		}
	}
}

// CORS is middleware that sets CORS headers.
func CORS(origins ...string) MiddlewareFunc {
	allowOrigin := "*"
	if len(origins) > 0 {
		allowOrigin = origins[0]
	}
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			ctx.SetHeader("Access-Control-Allow-Origin", allowOrigin)
			ctx.SetHeader("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			ctx.SetHeader("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if ctx.Method() == http.MethodOptions {
				ctx.Status(http.StatusNoContent).Text("")
				return nil
			}
			return next(ctx)
		}
	}
}

// Chain composes multiple middleware into one.
func Chain(middlewares ...MiddlewareFunc) MiddlewareFunc {
	return func(next Handler) Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}
