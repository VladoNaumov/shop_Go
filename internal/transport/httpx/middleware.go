package httpx

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self'; script-src 'self'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer-when-downgrade")
		// Включайте HSTS только за HTTPS-прокси:
		// w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		next.ServeHTTP(w, r)
	})
}

func requestLogger(next http.Handler) http.Handler {
	return middleware.Logger(next)
}

func commonMiddlewares(next http.Handler) http.Handler {
	h := next
	h = middleware.Compress(5)(h)
	h = middleware.Timeout(15 * time.Second)(h)
	h = middleware.Recoverer(h)
	h = middleware.RealIP(h)
	h = middleware.RequestID(h)
	h = requestLogger(h)
	h = secureHeaders(h)
	return h
}
