package app

// app.go — собирает приложение как в Echo: middleware + маршруты + статика + 404
// но на chi/net/http (без фреймворка).

import (
	"net/http"
	"time"

	"myApp/internal/core"
	"myApp/internal/http/handler"
	mw "myApp/internal/http/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/csrf"
)

func New(cfg core.Config, csrfKey []byte) http.Handler {
	r := chi.NewRouter()

	// --- middleware (как в Echo app.go) ---
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(15 * time.Second))
	r.Use(mw.SecureHeaders)       // CSP, XFO, Referrer, MIME, Permissions, COOP
	r.Use(mw.ServerKeepAliveHint) // необязательно: выставляет Keep-Alive

	// HSTS (только для HTTPS/прода)
	if cfg.Secure {
		r.Use(mw.HSTS)
	}

	// CSRF (gorilla) — токен доступен как csrf.TemplateField(r) в шаблонах
	r.Use(csrf.Protect(
		csrfKey,
		csrf.Secure(cfg.Secure),
		csrf.SameSite(csrf.SameSiteLaxMode),
		csrf.HttpOnly(true),
		csrf.Path("/"),
	))

	// --- статика ---
	static := http.FileServer(http.Dir("web/assets"))
	// кэш для продакшена
	if cfg.Env == "prod" {
		static = mw.CacheStatic(static)
	}
	r.Handle("/assets/*", http.StripPrefix("/assets/", static))

	// --- маршруты (как routes.go в Echo) ---
	r.Get("/", handler.Home)
	r.Get("/about", handler.About)
	r.Get("/form", handler.FormIndex)
	r.Post("/form", handler.FormSubmit)
	r.Get("/healthz", handler.Health)

	// --- 404 в самом конце ---
	r.NotFound(handler.NotFound)

	return r
}
