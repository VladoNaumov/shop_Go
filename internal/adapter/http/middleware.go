package httprouter

// Общие middleware: gzip, таймаут, recover, request id, real ip, логгер, безопасные заголовки.

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// secureHeaders добавляет безопасные заголовки HTTP и Content Security Policy (CSP)
func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CSP, совместимая с Bootstrap 5 (jsDelivr) + безопасные заголовки
		csp := strings.Join([]string{
			// Разрешаем ресурсы только с нашего домена
			"default-src 'self'",
			"base-uri 'self'",
			"object-src 'none'",

			// Изображения и data: (для favicon, иконок, placeholder)
			"img-src 'self' data:",

			// ✅ Разрешаем Bootstrap CSS с jsDelivr + (временно) 'unsafe-inline' для style-атрибутов
			"style-src 'self' https://cdn.jsdelivr.net 'unsafe-inline'",

			// ✅ Разрешаем Bootstrap JS с jsDelivr
			"script-src 'self' https://cdn.jsdelivr.net",

			// ✅ Разрешаем шрифты (font-face) и data: URI
			"font-src 'self' https://cdn.jsdelivr.net data:",

			// ✅ Разрешаем загрузку карт CSS/JS (source maps) из jsDelivr
			"connect-src 'self' https://cdn.jsdelivr.net",

			// ✅ Разрешаем только локальные формы
			"form-action 'self'",

			// ✅ Запрещаем встраивание в iframe
			"frame-ancestors 'none'",
		}, "; ")

		w.Header().Set("Content-Security-Policy", csp)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer-when-downgrade")

		next.ServeHTTP(w, r)
	})
}

// Лёгкий логгер chi (при желании можно заменить на slog со своими полями)
func requestLogger(next http.Handler) http.Handler {
	return middleware.Logger(next)
}

// commonMiddlewares объединяет стандартные middleware
func commonMiddlewares(next http.Handler) http.Handler {
	h := next
	h = middleware.Compress(5)(h)               // gzip-сжатие
	h = middleware.Timeout(15 * time.Second)(h) // таймаут на обработку
	h = middleware.Recoverer(h)                 // перехват паник
	h = middleware.RealIP(h)                    // определение реального IP
	h = middleware.RequestID(h)                 // уникальный ID запроса
	h = requestLogger(h)                        // логирование запросов
	h = secureHeaders(h)                        // безопасные заголовки + CSP
	return h
}
