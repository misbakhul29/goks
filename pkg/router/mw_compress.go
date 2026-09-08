package router

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// Compress is a middleware that compresses the HTTP response using Gzip
// if the client supports it via the Accept-Encoding header.
func Compress() MiddlewareFunc {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			if !strings.Contains(ctx.Request().Header.Get("Accept-Encoding"), "gzip") {
				return next(ctx)
			}
			
			ctx.SetHeader("Content-Encoding", "gzip")
			ctx.SetHeader("Vary", "Accept-Encoding")
			
			gz := gzip.NewWriter(ctx.Response())
			defer gz.Close()
			
			// Create a wrapped ResponseWriter that writes to the gzip writer
			gzw := gzipResponseWriter{
				ResponseWriter: ctx.Response(),
				Writer:         gz,
			}
			
			// We need to inject this wrapped response writer into the context.
			// Assuming Context has a field 'w' of type http.ResponseWriter.
			ctx.w = gzw
			
			return next(ctx)
		}
	}
}
