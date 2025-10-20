package core

// proxy.go

import (
	"net"
	"net/http"
	"strings"
)

// TrustedProxy валидирует, что запрос пришёл от доверенного прокси,
// и корректно выставляет схему из X-Forwarded-Proto.
// ВНИМАНИЕ: RemoteAddr НЕ меняем (формат должен быть host:port).
func TrustedProxy(trustedIPs []string) func(http.Handler) http.Handler {
	trusted := make(map[string]struct{}, len(trustedIPs))
	for _, ip := range trustedIPs {
		trusted[ip] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil || clientIP == "" {
				http.Error(w, "Неверный адрес клиента", http.StatusBadRequest)
				return
			}

			// Разрешаем только запросы, пришедшие с доверенного прокси (например, с loopback/Nginx)
			if _, ok := trusted[clientIP]; !ok {
				http.Error(w, "Недоверенный прокси", http.StatusForbidden)
				return
			}

			// Схема с прокси
			if proto := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))); proto == "https" {
				r.URL.Scheme = "https"
			} else {
				r.URL.Scheme = "http"
			}

			next.ServeHTTP(w, r)
		})
	}
}
