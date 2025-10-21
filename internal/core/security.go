package core

// security.go — отвечает за установку безопасных HTTP-заголовков.
// Включает Content-Security-Policy (CSP) с nonce, HSTS и другие меры защиты.

import (
	"github.com/gin-gonic/gin"
)

// -----------------------------------------------------------
// SecureHeaders — middleware: CSP с nonce + безопасные заголовки
// -----------------------------------------------------------
//
// Устанавливает основные защитные заголовки:
//
//  • Content-Security-Policy (CSP) — ограничивает источники JS, CSS, изображений.
//  • X-Content-Type-Options: nosniff — блокирует MIME-type sniffing.
//  • X-Frame-Options: DENY — запрещает встраивание страницы в iframe.
//  • Referrer-Policy — управляет передачей заголовка Referer.
//  • Permissions-Policy — запрещает доступ к камере, микрофону, геолокации и т.д.
//
// CSP использует nonce (одноразовый ключ на каждый запрос), чтобы разрешить
// выполнение только тех inline-стилей/скриптов, у которых атрибут nonce совпадает.

func SecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Извлекаем nonce из контекста запроса.
		nonce, ok := c.Request.Context().Value(CtxNonce).(string)
		if !ok || nonce == "" {
			// Если nonce не найден — это ошибка и потенциально небезопасное состояние.
			FailC(c, Internal("Nonce не найден", nil))
			return
		}

		// Формируем Content-Security-Policy.
		// Разрешаем только:
		//   - свои ресурсы ('self')
		//   - data: для изображений и шрифтов
		//   - jsdelivr.net для внешних библиотек (CSS/JS/шрифты)
		//   - инлайн-скрипты/стили только с nonce, выданным для этого запроса.
		csp := "default-src 'self'; " +
			"img-src 'self' data:; " + // изображения только локальные или data:
			"style-src 'self' https://cdn.jsdelivr.net 'nonce-" + nonce + "'; " + // стили из jsdelivr и с nonce
			"script-src 'self' https://cdn.jsdelivr.net 'nonce-" + nonce + "'; " + // JS — локально, CDN, с nonce
			"font-src 'self' https://cdn.jsdelivr.net data:; " + // шрифты локальные, CDN, data:
			"connect-src 'self' https://cdn.jsdelivr.net; " + // разрешены запросы к jsdelivr (напр. sourcemaps)
			"form-action 'self'; " + // формы можно отправлять только на свой домен
			"frame-ancestors 'none'; " + // запрещаем встраивание в iframe
			"base-uri 'self'" // запрет подмены <base href>

		// Устанавливаем заголовки безопасности.
		c.Writer.Header().Set("Content-Security-Policy", csp)

		// Блокирует попытки браузера угадать MIME-тип (XSS-защита).
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")

		// Запрещает встраивать сайт в iframe (Clickjacking-защита).
		c.Writer.Header().Set("X-Frame-Options", "DENY")

		// Управляет передачей Referer: будет отправляться только домен и протокол.
		c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Ограничивает доступ к чувствительным возможностям браузера.
		c.Writer.Header().Set("Permissions-Policy",
			"camera=(), microphone=(), geolocation=(), payment=()")

		// Передаём управление дальше.
		c.Next()
	}
}

// -----------------------------------------------------------
// HSTS — включает Strict-Transport-Security для HTTPS
// -----------------------------------------------------------
//
// HSTS (HTTP Strict Transport Security) заставляет браузер всегда
// использовать HTTPS при повторных запросах на домен.
//
//  • Работает ТОЛЬКО при secure-режиме (в продакшене).
//  • "max-age=31536000" — браузер будет помнить правило 1 год.
//  • "includeSubDomains" — правило распространяется на поддомены.
//
// При dev-режиме (без HTTPS) этот заголовок не устанавливается,
// чтобы не мешать локальной отладке.

func HSTS(isProduction bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isProduction {
			// В режиме разработки ничего не добавляем.
			c.Next()
			return
		}

		// Устанавливаем HSTS только при HTTPS в продакшене.
		c.Writer.Header().Set(
			"Strict-Transport-Security",
			"max-age=31536000; includeSubDomains",
		)

		c.Next()
	}
}

/*

### 🧠 Краткое объяснение, зачем нужны эти заголовки

| Заголовок                            | Назначение                                                      | Пример значения                                                                 |
| ------------------------------------ | --------------------------------------------------------------- | ------------------------------------------------------------------------------- |
| **Content-Security-Policy**          | Контроль разрешённых источников контента (CSS, JS, изображения) | `"default-src 'self'; script-src 'self' https://cdn.jsdelivr.net 'nonce-XYZ';"` |
| **X-Content-Type-Options**           | Запрещает MIME type sniffing (XSS-защита)                       | `"nosniff"`                                                                     |
| **X-Frame-Options**                  | Запрещает встраивание сайта в iframe                            | `"DENY"`                                                                        |
| **Referrer-Policy**                  | Управляет, что передаётся в Referer при переходах               | `"strict-origin-when-cross-origin"`                                             |
| **Permissions-Policy**               | Запрещает доступ к камере, микрофону, геолокации                | `"camera=(), microphone=(), geolocation=()"`                                    |
| **Strict-Transport-Security (HSTS)** | Обязывает браузер использовать HTTPS                            | `"max-age=31536000; includeSubDomains"`                                         |

*/
