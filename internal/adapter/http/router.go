package httprouter

/*
Определяет маршруты: /, /healthz, /readyz, /metrics, /assets/*,
подключает middleware и обработчики ошибок 404/405.
*/

import (
	"net/http"
	"os"
	"path/filepath"

	"example.com/shop/internal/adapter/http/handler"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter() http.Handler {
	r := chi.NewRouter()

	// middleware
	r.Use(commonMiddlewares)

	// базовые маршруты
	r.Get("/", handler.HomeIndex)
	r.Get("/healthz", handler.Health)
	r.Get("/readyz", handler.Ready)
	r.Handle("/metrics", promhttp.Handler())
	r.Get("/about", handler.About)

	// статика
	assetsDir := http.Dir(filepath.Clean("web/assets"))
	assetsFS := http.FileServer(assetsDir)
	r.Handle("/assets/*", http.StripPrefix("/assets/", cacheStatic(assetsFS)))

	// 404 / 405
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
		// dev — короткий кэш, prod — длинный + хэши в именах файлов
		w.Header().Set("Cache-Control", "public, max-age=300")
		next.ServeHTTP(w, r)
	})
}

func isDev() bool {
	return os.Getenv("APP_ENV") == "dev" || os.Getenv("APP_ENV") == ""
}
