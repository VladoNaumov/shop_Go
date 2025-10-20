package core

// security.go
import (
	"net/http"
)

// SecureHeaders добавляет заголовки безопасности, включая CSP с nonce из контекста ( Security Misconfiguration)
func SecureHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Получает nonce из контекста запроса
			nonce, ok := r.Context().Value(CtxNonce).(string)
			if !ok || nonce == "" {
				Fail(w, r, Internal("Nonce не найден в контексте", nil))
				return
			}

			// Настраивает Content Security Policy с поддержкой nonce
			csp := "" +
				"default-src 'self'; " +
				"img-src 'self' storage:; " +
				"style-src 'self' https://cdn.jsdelivr.net 'unsafe-inline' 'nonce-" + nonce + "'; " +
				"script-src 'self' https://cdn.jsdelivr.net 'nonce-" + nonce + "'; " +
				"font-src 'self' https://cdn.jsdelivr.net storage:; " +
				"connect-src 'self' https://cdn.jsdelivr.net; " +
				"form-action 'self'; " +
				"frame-ancestors 'none'; " +
				"base-uri 'self'"

			// Устанавливает заголовки безопасности
			w.Header().Set("Content-Security-Policy", csp)
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
			w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")

			next.ServeHTTP(w, r)
		})
	}
}

// HSTS включает Strict-Transport-Security для продакшен-среды (Cryptographic Failures)
func HSTS(isProduction bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Добавляет HSTS-заголовок в продакшене
			if isProduction {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
			}
			next.ServeHTTP(w, r)
		})
	}
}
