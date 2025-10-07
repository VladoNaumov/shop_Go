// security.go
package middleware

import (
	"net/http"
	"strings"
)

func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		csp := strings.Join([]string{
			"default-src 'self'",
			"img-src 'self' data:",
			"style-src 'self' https://cdn.jsdelivr.net 'unsafe-inline'",
			"script-src 'self' https://cdn.jsdelivr.net",
			"font-src 'self' https://cdn.jsdelivr.net data:",
			"connect-src 'self' https://cdn.jsdelivr.net",
			"form-action 'self'",
			"frame-ancestors 'none'",
		}, "; ")
		w.Header().Set("Content-Security-Policy", csp)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer-when-downgrade")
		next.ServeHTTP(w, r)
	})
}
