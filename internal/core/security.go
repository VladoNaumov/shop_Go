package core

// security.go
import (
	"net/http"

	"github.com/unrolled/secure"
)

// SecureHeaders добавляет заголовки безопасности, включая CSP с nonce
func SecureHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Получаем nonce из контекста
			nonce, ok := r.Context().Value(CtxNonce).(string)
			if !ok || nonce == "" {
				Fail(w, r, Internal("Nonce не найден в контексте", nil))
				return
			}

			// Формируем CSP с подстановкой nonce
			csp := "default-src 'self'; " +
				"img-src 'self' storage:; " +
				"style-src 'self' https://cdn.jsdelivr.net 'unsafe-inline' 'nonce-" + nonce + "'; " +
				"script-src 'self' https://cdn.jsdelivr.net 'nonce-" + nonce + "'; " +
				"font-src 'self' https://cdn.jsdelivr.net storage:; " +
				"connect-src 'self' https://cdn.jsdelivr.net; " +
				"form-action 'self'; " +
				"frame-ancestors 'none'; " +
				"base-uri 'self'"

			// Настраиваем secure middleware
			secureMiddleware := secure.New(secure.Options{
				ContentSecurityPolicy: csp,
				FrameDeny:             true,
				ContentTypeNosniff:    true,
				BrowserXssFilter:      true,
				ReferrerPolicy:        "strict-origin-when-cross-origin",
				PermissionsPolicy:     "camera=(), microphone=(), geolocation=(), payment=()",
				IsDevelopment:         false, // Установите true для разработки
			})

			secureMiddleware.Handler(next).ServeHTTP(w, r)
		})
	}
}

// HSTS включает Strict-Transport-Security для продакшен-среды
func HSTS(isProduction bool) func(http.Handler) http.Handler {
	secureMiddleware := secure.New(secure.Options{
		STSSeconds:           31536000,
		STSIncludeSubdomains: true,
		STSPreload:           true,
		IsDevelopment:        !isProduction,
	})

	return secureMiddleware.Handler
}
