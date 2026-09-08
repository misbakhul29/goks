package router

// Secure is a middleware that adds basic security headers to the response.
// It helps protect against Cross-Site Scripting (XSS), Clickjacking, and other attacks.
func Secure() MiddlewareFunc {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			// Prevents the browser from rendering the page inside a frame/iframe
			ctx.SetHeader("X-Frame-Options", "DENY")
			
			// Enables XSS filtering in the browser
			ctx.SetHeader("X-XSS-Protection", "1; mode=block")
			
			// Prevents MIME-sniffing
			ctx.SetHeader("X-Content-Type-Options", "nosniff")
			
			// Enforces HTTPS (Strict-Transport-Security) for 1 year
			ctx.SetHeader("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			
			// Controls how much referrer information should be included with requests
			ctx.SetHeader("Referrer-Policy", "strict-origin-when-cross-origin")
			
			return next(ctx)
		}
	}
}
