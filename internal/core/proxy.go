package core

import (
	"net"
	"net/http"
	"strings"
)

// TrustedProxy валидирует, что запрос пришёл от доверенного прокси,
func TrustedProxy(trustedIPs []string) func(http.Handler) http.Handler {
	trusted := make([]*net.IPNet, 0, len(trustedIPs))
	for _, ipStr := range trustedIPs {
		_, ipNet, err := net.ParseCIDR(ipStr)
		if err != nil {
			// Для одиночных IP
			ip := net.ParseIP(ipStr)
			if ip != nil {
				ipNet = &net.IPNet{IP: ip, Mask: net.CIDRMask(32*len(ip), 32*len(ip))}
			}
		}
		if ipNet != nil {
			trusted = append(trusted, ipNet)
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil || clientIP == "" {
				http.Error(w, "Неверный адрес клиента", http.StatusBadRequest)
				return
			}

			ip := net.ParseIP(clientIP)
			if ip == nil {
				http.Error(w, "Неверный IP", http.StatusBadRequest)
				return
			}

			isTrusted := false
			for _, ipNet := range trusted {
				if ipNet.Contains(ip) {
					isTrusted = true
					break
				}
			}
			if !isTrusted {
				http.Error(w, "Недоверенный прокси", http.StatusForbidden)
				return
			}

			proto := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")))
			if proto == "https" {
				r.URL.Scheme = "https"
			} else {
				r.URL.Scheme = "http"
			}

			next.ServeHTTP(w, r)
		})
	}
}
