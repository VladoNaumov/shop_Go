package core

// common.go — базовые middleware приложения.
// Используется стандартный логгер chi + recoverer + timeout + безопасные заголовки.

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// UseCommon подключает общие middleware для всех маршрутов.
// Порядок подключения важен.
func UseCommon(r *chi.Mux) {
	// 1) Присваивает каждому запросу уникальный ID
	r.Use(middleware.RequestID)

	// 2) Определяет реальный IP клиента (X-Forwarded-For / X-Real-IP)
	r.Use(middleware.RealIP)

	// 3) Логирование всех запросов (в stdout)
	r.Use(middleware.Logger)

	// 4) Восстановление после паники, чтобы сервер не падал
	r.Use(middleware.Recoverer)

	// 5) Таймаут выполнения запроса (например, 15 секунд)
	r.Use(middleware.Timeout(15 * time.Second))

	// 6) Безопасные заголовки HTTP (CSP, XFO, MIME, Referrer и др.)
	r.Use(SecureHeaders)
}
