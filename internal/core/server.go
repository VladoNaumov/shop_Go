package core

import (
	"net/http"
	"time"
)

// Server возвращает готовую конфигурацию http.Server.
// Эта функция задаёт "безопасные" значения таймаутов,
// чтобы сервер не зависал на медленных клиентах.
func Server(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:    addr,    // Адрес, на котором будет слушать сервер (например, ":8080")
		Handler: handler, // Главный обработчик запросов (роутер + middleware)

		// --- Таймауты для защиты от "медленных" клиентов ---

		ReadHeaderTimeout: 5 * time.Second,  // максимум 5 сек на чтение HTTP-заголовков запроса
		ReadTimeout:       10 * time.Second, // максимум 10 сек на полное чтение тела запроса
		WriteTimeout:      30 * time.Second, // максимум 30 сек на запись ответа клиенту
		IdleTimeout:       60 * time.Second, // сколько держать соединение открытым (keep-alive)
	}
}
