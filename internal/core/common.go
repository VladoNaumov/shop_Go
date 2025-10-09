package core

// common.go
import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// UseCommon подключает базовые (общие) middleware для всего приложения.
// Эти middleware обрабатывают логи, IP-адреса, восстановление после паник,
// ограничение по времени, и добавляют базовые заголовки безопасности.
func UseCommon(r *chi.Mux) {
	// --- Идентификатор запроса ---
	// Добавляет уникальный Request ID к каждому запросу.
	// Это помогает связывать логи между фронтом и беком.
	r.Use(middleware.RequestID)

	// --- Определение реального IP клиента ---
	// Извлекает IP из заголовков X-Forwarded-For / X-Real-IP,
	// что полезно при работе за reverse-proxy (Nginx, Cloudflare и т.д.).
	r.Use(middleware.RealIP)

	// --- Логирование ---
	// Логирует метод, путь, статус, длительность и т.д.
	r.Use(middleware.Logger)

	// --- Восстановление после паники ---
	// Перехватывает паники внутри хендлеров и возвращает 500 вместо падения сервера.
	r.Use(middleware.Recoverer)

	// --- Таймаут запроса ---
	// Прерывает запрос, если он выполняется дольше 15 секунд.
	// Помогает защититься от зависших хендлеров или медленных клиентов.
	r.Use(middleware.Timeout(15 * time.Second))

	// --- Безопасные заголовки ---
	// Подключает собственный middleware SecureHeaders (Content-Security-Policy и др.)
	// Реализуется в другом файле, например secure_headers.go.
	r.Use(SecureHeaders)
}
