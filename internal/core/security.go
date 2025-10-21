package core

// security.go
import (
	"github.com/gin-contrib/secure"
	"github.com/gin-gonic/gin"
)

// SecureHeaders — Gin-middleware: собирает CSP с nonce и проставляет безопасность.
// В gin-contrib/secure нет динамического поля для CSP -> ставим CSP вручную.
func SecureHeaders() gin.HandlerFunc {
	sec := secure.New(secure.Config{
		FrameDeny:          true,
		ContentTypeNosniff: true,
		BrowserXssFilter:   true,
		ReferrerPolicy:     "strict-origin-when-cross-origin",
	})

	return func(c *gin.Context) {
		// Достаём nonce либо из Gin-контекста (рекомендуется),
		nonce := c.GetString("nonce")
		if nonce == "" {
			if v := c.Request.Context().Value(CtxNonce); v != nil {
				if s, ok := v.(string); ok {
					nonce = s
				}
			}
		}
		if nonce == "" {
			FailC(c, Internal("Nonce не найден в контексте", nil))
			return
		}

		sec(c)

		// Динамический CSP с nonce:
		csp := "default-src 'self'; " +
			"img-src 'self' storage:; " +
			"style-src 'self' https://cdn.jsdelivr.net 'unsafe-inline' 'nonce-" + nonce + "'; " +
			"script-src 'self' https://cdn.jsdelivr.net 'nonce-" + nonce + "'; " +
			"font-src 'self' https://cdn.jsdelivr.net storage:; " +
			"connect-src 'self' https://cdn.jsdelivr.net; " +
			"form-action 'self'; " +
			"frame-ancestors 'none'; " +
			"base-uri 'self'"

		c.Writer.Header().Set("Content-Security-Policy", csp)

		// В gin-contrib/secure НЕТ поля PermissionsPolicy -> ставим вручную:
		c.Writer.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")

		c.Next()
	}
}

// HSTS — включает Strict-Transport-Security
func HSTS(isProduction bool) gin.HandlerFunc {
	return secure.New(secure.Config{
		STSSeconds:           31536000,
		STSIncludeSubdomains: true,
		IsDevelopment:        !isProduction,
	})
}
