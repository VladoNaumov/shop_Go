package httpx

// Определяет маршруты: /, /healthz, /readyz, /metrics, /assets/*,
// подключает middleware и обработчики ошибок 404/405.

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"example.com/shop/internal/transport/httpx/handlers"
)

func NewRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(commonMiddlewares)

	// Health/Ready (готовность пока без БД)
	r.Get("/healthz", handlers.Health)
	r.Get("/readyz", handlers.Ready)

	// Prometheus метрики
	r.Handle("/metrics", promhttp.Handler())

	// Статика (только GET) + кэш
	assetsDir := http.Dir(filepath.Clean("web/assets"))
	fs := http.FileServer(assetsDir)
	r.Group(func(r chi.Router) {
		r.Get("/assets/*", http.StripPrefix("/assets/", cacheStatic(fs)).ServeHTTP)
	})

	// Главная (SSR)
	r.Get("/", handlers.HomeIndex) // аналог Controller@index

	// Стандартизированные ответы
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	return r
}

func cacheStatic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// В dev короткий кэш; в prod — длинный с хэшами в именах
		w.Header().Set("Cache-Control", "public, max-age=300")
		next.ServeHTTP(w, r)
	})
}

// Утилита (может пригодиться для фич по окружению)
func isDev() bool {
	return os.Getenv("APP_ENV") == "dev" || os.Getenv("APP_ENV") == ""
}
