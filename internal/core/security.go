package core

// security.go
import (
	"github.com/gin-gonic/gin"
)

// SecureHeaders — middleware: CSP с nonce + безопасные заголовки
func SecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		nonce, ok := c.Request.Context().Value(CtxNonce).(string)
		if !ok || nonce == "" {
			FailC(c, Internal("Nonce не найден", nil))
			return
		}

		csp := "default-src 'self'; " +
			"img-src 'self' data:; " +
			"style-src 'self' https://cdn.jsdelivr.net 'nonce-" + nonce + "'; " +
			"script-src 'self' https://cdn.jsdelivr.net 'nonce-" + nonce + "'; " +
			"font-src 'self' https://cdn.jsdelivr.net data:; " +
			"connect-src 'self' https://cdn.jsdelivr.net; " + // ← РАЗРЕШЕНО: .map файлы
			"form-action 'self'; " +
			"frame-ancestors 'none'; " +
			"base-uri 'self'"

		c.Writer.Header().Set("Content-Security-Policy", csp)
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Writer.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")

		c.Next()
	}
}

// HSTS — включает Strict-Transport-Security
func HSTS(isProduction bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isProduction {
			c.Next()
			return
		}
		c.Writer.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Next()
	}
}
