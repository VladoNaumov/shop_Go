package middleware

import "net/http"

// SecureHeaders — безопасные HTTP-заголовки (CSP, XFO, Referrer, MIME, Permissions, COOP).
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", ""+
			"default-src 'self'; "+
			"img-src 'self' data:; "+
			"style-src 'self' https://cdn.jsdelivr.net 'unsafe-inline'; "+
			"script-src 'self' https://cdn.jsdelivr.net; "+
			"font-src 'self' https://cdn.jsdelivr.net data:; "+
			"connect-src 'self' https://cdn.jsdelivr.net; "+
			"form-action 'self'; "+
			"frame-ancestors 'none'")

		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer-when-downgrade")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")

		next.ServeHTTP(w, r)
	})
}

// HSTS — Strict-Transport-Security (включать только при HTTPS/проде).
func HSTS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		next.ServeHTTP(w, r)
	})
}

// CacheStatic — добавляет заголовки кэширования для статики (для prod).
func CacheStatic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("Vary", "Accept-Encoding")
		next.ServeHTTP(w, r)
	})
}

// ServerKeepAliveHint — необязательный заголовок для наглядности.
func ServerKeepAliveHint(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Keep-Alive", "timeout=60")
		next.ServeHTTP(w, r)
	})
}
