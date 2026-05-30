package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders sets baseline browser-hardening response headers on every
// request. The strict transport + CSP headers are only emitted when prod is true
// (they assume HTTPS and would break a plain-HTTP dev setup).
func SecurityHeaders(prod bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("X-XSS-Protection", "0") // modern browsers: rely on CSP, disable legacy auditor
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		if prod {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			// API responses are JSON; a restrictive default-src is safe here. The SPA
			// is served separately (nginx) and should ship its own CSP for its assets.
			h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		}
		c.Next()
	}
}
