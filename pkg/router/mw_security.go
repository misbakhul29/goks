package router

// Secure is a middleware that adds basic security headers to the response.
// It helps protect against Cross-Site Scripting (XSS), Clickjacking, and other attacks.
func Secure() MiddlewareFunc {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			// Prevents the browser from rendering the page inside a frame/iframe
			ctx.SetHeader("X-Frame-Options", "DENY")

			// Disables legacy buggy XSS auditor in modern browsers (standard modern best practice)
			ctx.SetHeader("X-XSS-Protection", "0")

			// Prevents MIME-sniffing
			ctx.SetHeader("X-Content-Type-Options", "nosniff")

			// Enforces HTTPS (Strict-Transport-Security) for 1 year when served over HTTPS
			if ctx.Request().TLS != nil || ctx.Header("X-Forwarded-Proto") == "https" {
				ctx.SetHeader("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			// Controls how much referrer information should be included with requests
			ctx.SetHeader("Referrer-Policy", "strict-origin-when-cross-origin")

			// Restricts access to sensitive browser features by default
			ctx.SetHeader("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

			return next(ctx)
		}
	}
}
