package httpx

import (
	"net/http"

	"myApp/internal/http/handler"
	"myApp/internal/http/middleware"

	"github.com/go-chi/chi/v5"
)

func NewRouter() http.Handler {
	r := chi.NewRouter()
	middleware.UseCommon(r) // gzip, recover, timeout, csp, logger

	// страницы
	r.Get("/", handler.Home)
	r.Get("/about", handler.About)
	r.Get("/form", handler.FormIndex)
	r.Post("/form", handler.FormSubmit)

	// статика (если нужна)
	// r.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.Dir("web/assets"))))

	// health
	r.Get("/healthz", handler.Health)
	r.NotFound(handler.NotFound)
	return r
}
