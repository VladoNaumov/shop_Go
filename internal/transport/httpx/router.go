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

func Router() http.Handler {
	// Создаём новый роутер из пакета chi
	router := chi.NewRouter()
	// Подключение middleware
	router.Use(commonMiddlewares)

	// Основные маршруты
	router.Get("/", handlers.HomeIndex)
	router.Get("/healthz", handlers.Health)
	router.Get("/readyz", handlers.Ready)
	router.Handle("/metrics", promhttp.Handler()) //Отдаёт метрики Prometheus

	// Раздача статических файлов
	// http.Dir — создаёт виртуальную директорию, где будут искаться файлы.
	assetsDir := http.Dir(filepath.Clean("web/assets"))
	// http.FileServer — превращает эту директорию в обработчик (умеет отдавать .css, .js, .png и т.п.).
	fs := http.FileServer(assetsDir)

	// создаёт подгруппу маршрутов
	router.Group(func(r chi.Router) {
		r.Get("/assets/*", http.StripPrefix("/assets/", cacheStatic(fs)).ServeHTTP)
	})

	// Обработчики ошибок
	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	return router // Теперь Router() можно подключить в main.go -> http.ListenAndServe(":8080", Router())
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
