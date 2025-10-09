package core

// security.go
import (
	"net/http"
	"strings"
)

// SecureHeaders — middleware, добавляющий стандартные HTTP-заголовки безопасности.
// Он защищает приложение от XSS, кликджекинга, утечки рефереров и MIME-подмены.
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// --- Content Security Policy (CSP) ---
		// Задаёт белый список разрешённых источников контента (JS, CSS, изображения и т.д.)
		// Это защищает от внедрения вредоносного кода (XSS).
		csp := strings.Join([]string{
			"default-src 'self'",   // всё по умолчанию — только с этого домена
			"img-src 'self' data:", // разрешаем изображения и data: (для inline base64)
			"style-src 'self' https://cdn.jsdelivr.net 'unsafe-inline'", // стили: локальные и с CDN (разрешаем inline CSS)
			"script-src 'self' https://cdn.jsdelivr.net",                // JS: только локально и с CDN
			"font-src 'self' https://cdn.jsdelivr.net data:",            // шрифты: локальные, CDN и data:
			"connect-src 'self' https://cdn.jsdelivr.net",               // разрешённые сетевые запросы (fetch, XHR)
			"form-action 'self'",                                        // формы могут отправляться только на тот же домен
			"frame-ancestors 'none'",                                    // запрещает встраивание сайта в iframe
		}, "; ")

		// Устанавливаем CSP-заголовок
		w.Header().Set("Content-Security-Policy", csp)

		// --- X-Content-Type-Options ---
		// Запрещает браузеру "угадывать" MIME-тип (предотвращает MIME-sniffing атаки)
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// --- X-Frame-Options ---
		// Запрещает встраивать страницу в <iframe> (защита от кликджекинга)
		w.Header().Set("X-Frame-Options", "DENY")

		// --- Referrer-Policy ---
		// Контролирует, какие данные об источнике запроса (Referer) передаются на внешний сайт
		// "no-referrer-when-downgrade" — не передаёт реферер при переходе с HTTPS на HTTP
		w.Header().Set("Permissions-Policy",
			"camera=(), microphone=(), geolocation=(), payment=()")

		// Изоляция контекста страницы (минимум — защита от некоторых XS-Leaks)
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")

		// Передаём управление следующему обработчику
		next.ServeHTTP(w, r)

	})
}
