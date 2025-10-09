package core

// security.go — безопасные HTTP-заголовки для всего приложения.
// Здесь ставим CSP, защиту от кликджекинга, запрет MIME-sniffing,
// политику реферера, ограничения по API браузера и COOP.
//
// ⚠️ В продакшене не дублируем HSTS здесь, т.к. ты уже добавляешь его в main.go
// при cfg.Secure (это ок — так проще управлять через конфиг).

import (
	"net/http"
	"strings"
)

// SecureHeaders — middleware, добавляющее стандартные HTTP-заголовки безопасности.
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// --- Content Security Policy (CSP) ---
		// Белые списки источников для разных типов ресурсов.
		// При необходимости дополни домены/CDN под свой фронтенд.
		csp := strings.Join([]string{
			"default-src 'self'",   // всё по умолчанию — только с текущего домена
			"img-src 'self' data:", // изображения и data: (base64)
			"style-src 'self' https://cdn.jsdelivr.net 'unsafe-inline'", // стили: локальные + CDN; inline-css разрешён (если нужно)
			"script-src 'self' https://cdn.jsdelivr.net",                // скрипты: локальные + CDN
			"font-src 'self' https://cdn.jsdelivr.net data:",            // шрифты: локальные + CDN + data:
			"connect-src 'self' https://cdn.jsdelivr.net",               // fetch/XHR/WebSocket источники
			"form-action 'self'",                                        // отправка форм только на текущий домен
			"frame-ancestors 'none'",                                    // запрет встраивания в <iframe> (кликджекинг)
		}, "; ")
		w.Header().Set("Content-Security-Policy", csp)

		// --- X-Content-Type-Options ---
		// Запрещает браузеру "угадывать" MIME-тип (nosniff).
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// --- X-Frame-Options ---
		// Защита от кликджекинга (запрет встраивания в iframe).
		w.Header().Set("X-Frame-Options", "DENY")

		// --- Referrer-Policy ---
		// Управляет тем, какой реферер (Referer) уходит на внешние сайты.
		w.Header().Set("Referrer-Policy", "no-referrer-when-downgrade")

		// --- Permissions-Policy ---
		// Ограничивает доступ к API/возможностям браузера (камера/микрофон/геолокация/оплата).
		w.Header().Set("Permissions-Policy",
			"camera=(), microphone=(), geolocation=(), payment=()")

		// --- Cross-Origin-Opener-Policy (COOP) ---
		// Изоляция browsing context (защита от некоторых XS-Leaks).
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")

		// Передаём управление дальше
		next.ServeHTTP(w, r)
	})
}
