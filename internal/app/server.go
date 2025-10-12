package app

//server.go
import (
	"context"
	"net/http"
	"time"

	"myApp/internal/core"
)

// Server создаёт HTTP-сервер с заданными таймаутами и обработчиком (OWASP A05: Security Misconfiguration)
func Server(cfg core.Config, h http.Handler) (*http.Server, error) {
	return &http.Server{
		Addr:              cfg.Addr,              // Адрес сервера
		Handler:           h,                     // Обработчик запросов
		ReadHeaderTimeout: cfg.ReadHeaderTimeout, // Таймаут чтения заголовков
		ReadTimeout:       cfg.ReadTimeout,       // Таймаут чтения запроса
		WriteTimeout:      cfg.WriteTimeout,      // Таймаут записи ответа
		IdleTimeout:       cfg.IdleTimeout,       // Таймаут простоя
	}, nil
}

// Shutdown выполняет graceful shutdown сервера с указанным таймаутом (OWASP A05)
func Shutdown(srv *http.Server, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return srv.Shutdown(ctx)
}
