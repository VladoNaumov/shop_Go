package httpx

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

	// health/ready
	r.Get("/healthz", handlers.Health)
	r.Get("/readyz", handlers.Ready)

	// метрики (позже можно закрыть basic auth или ip-filter за прокси)
	r.Handle("/metrics", promhttp.Handler())

	// статика (Cache-Control для ассетов)
	assetsDir := http.Dir(filepath.Clean("web/assets"))
	fs := http.FileServer(assetsDir)
	r.Handle("/assets/*", http.StripPrefix("/assets/", cacheStatic(fs)))

	// страницы
	r.Get("/", handlers.Home)

	return r
}

func cacheStatic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// для dev — короткий кэш, в prod можно выставить длительный (с хешами в именах файлов)
		w.Header().Set("Cache-Control", "public, max-age=300")
		next.ServeHTTP(w, r)
	})
}

// утилита (например, пригодится, если захотите определить окружение)
func isDev() bool {
	return os.Getenv("APP_ENV") == "dev" || os.Getenv("APP_ENV") == ""
}
