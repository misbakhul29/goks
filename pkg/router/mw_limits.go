package router

import (
	"bytes"
	"context"
	"net/http"
	"sync"
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

			origW := ctx.Response()
			tw := &timeoutWriter{ResponseWriter: origW, code: http.StatusOK}
			ctx.w = tw

			req := ctx.Request().WithContext(reqCtx)
			ctx.SetRequest(req)

			type result struct {
				err   error
				panic any
			}
			done := make(chan result, 1)

			go func() {
				defer func() {
					if r := recover(); r != nil {
						done <- result{panic: r}
					}
				}()
				err := next(ctx)
				done <- result{err: err}
			}()

			select {
			case <-reqCtx.Done():
				tw.markTimeout()
				origW.WriteHeader(http.StatusGatewayTimeout)
				_, _ = origW.Write([]byte("Gateway Timeout"))
				return nil
			case res := <-done:
				if res.panic != nil {
					panic(res.panic)
				}
				ctx.w = origW
				tw.flush(origW)
				return res.err
			}
		}
	}
}

type timeoutWriter struct {
	http.ResponseWriter
	mu          sync.Mutex
	buf         bytes.Buffer
	code        int
	wroteHeader bool
	timedOut    bool
}

func (tw *timeoutWriter) Header() http.Header {
	return tw.ResponseWriter.Header()
}

func (tw *timeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut || tw.wroteHeader {
		return
	}
	tw.code = code
	tw.wroteHeader = true
}

func (tw *timeoutWriter) Write(b []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut {
		return len(b), nil
	}
	return tw.buf.Write(b)
}

func (tw *timeoutWriter) flush(w http.ResponseWriter) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut {
		return
	}
	if tw.wroteHeader && tw.code > 0 {
		w.WriteHeader(tw.code)
	}
	_, _ = w.Write(tw.buf.Bytes())
}

func (tw *timeoutWriter) markTimeout() {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.timedOut = true
}
