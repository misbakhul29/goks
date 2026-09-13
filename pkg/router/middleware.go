// Package router - Common middleware for GoKS.
package router

import (
	"log"
	"net/http"
	"time"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// Logger is middleware that logs each request with method, path, status, and duration.
func Logger() MiddlewareFunc {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: ctx.w, status: ctx.status}
			ctx.w = sw
			err := next(ctx)
			status := sw.status
			if status == 0 {
				status = ctx.status
			}
			log.Printf("[GoKS] %d %s %s %v", status, ctx.Method(), ctx.Path(), time.Since(start))
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
// If origins are specified, it validates against the incoming Origin header and sets Vary: Origin.
// If no origins are provided, wildcard "*" is used.
func CORS(origins ...string) MiddlewareFunc {
	allowAll := len(origins) == 0
	allowed := make(map[string]bool)
	for _, o := range origins {
		if o == "*" {
			allowAll = true
		}
		allowed[o] = true
	}

	return func(next Handler) Handler {
		return func(ctx *Context) error {
			origin := ctx.Header("Origin")
			if allowAll {
				ctx.SetHeader("Access-Control-Allow-Origin", "*")
			} else if origin != "" && allowed[origin] {
				ctx.SetHeader("Access-Control-Allow-Origin", origin)
				ctx.SetHeader("Vary", "Origin")
			}

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
