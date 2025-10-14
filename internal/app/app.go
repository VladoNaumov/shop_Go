package app

// app.go
import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"myApp/internal/core"
	"myApp/internal/http/handler"
	mw "myApp/internal/http/middleware"
	"myApp/internal/view"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/csrf"
)

// New — собирает HTTP-роутер из модулей (middleware, маршруты, шаблоны и т.д.)
func New(cfg core.Config, csrfKey []byte) (http.Handler, error) {
	tpl, err := initTemplates()
	if err != nil {
		return nil, err
	}

	r := chi.NewRouter()
	useBaseMiddleware(r, cfg)
	useSecurityMiddleware(r, cfg)
	useCSRF(r, cfg, csrfKey)
	serveStatic(r)
	registerRoutes(r, tpl)

	return r, nil
}

/* ---------- Инициализация ---------- */

// initTemplates — создаёт экземпляр шаблонизатора
func initTemplates() (*view.Templates, error) {
	return view.New()
}

/* ---------- Middleware ---------- */

// useBaseMiddleware — подключает базовые middleware (лог, ID, таймауты, nonce и т.д.)
func useBaseMiddleware(r *chi.Mux, cfg core.Config) {
	r.Use(withNonce()) // добавляет nonce для CSP
	r.Use(mw.TrustedProxy([]string{"127.0.0.1", "::1"}))
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(cfg.RequestTimeout))
}

// useSecurityMiddleware — подключает CSP, X-Frame-Options, HSTS (при HTTPS)
func useSecurityMiddleware(r *chi.Mux, cfg core.Config) {
	r.Use(mw.SecureHeaders())
	if cfg.Secure {
		r.Use(mw.HSTS(cfg.Env == "prod"))
	}
}

// useCSRF — включает CSRF-защиту для форм
func useCSRF(r *chi.Mux, cfg core.Config, csrfKey []byte) {
	r.Use(csrf.Protect(
		csrfKey,
		csrf.Secure(cfg.Secure),
		csrf.SameSite(csrf.SameSiteLaxMode),
		csrf.HttpOnly(true),
		csrf.Path("/"),
	))
	if !cfg.Secure && cfg.Env != "prod" {
		core.LogError("CSRF работает без HTTPS в не-продакшен среде", nil)
	}
}

// withNonce — создаёт middleware, добавляющее случайный nonce в контекст запроса
func withNonce() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nonce, err := generateNonce()
			if err != nil {
				core.LogError("Ошибка генерации nonce", map[string]interface{}{"error": err.Error()})
				core.Fail(w, r, core.Internal("Ошибка генерации nonce", err))
				return
			}
			ctx := context.WithValue(r.Context(), core.CtxNonce, nonce)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

/* ---------- Статика и маршруты ---------- */

// serveStatic — обслуживает файлы из каталога web/assets
func serveStatic(r *chi.Mux) {
	fs := http.StripPrefix("/assets/", http.FileServer(http.Dir("web/assets")))
	r.Handle("/assets/*", fs)
}

// registerRoutes — регистрирует маршруты приложения
func registerRoutes(r *chi.Mux, tpl *view.Templates) {
	r.Get("/", handler.Home(tpl))
	r.Get("/about", handler.About(tpl))
	r.Get("/form", handler.FormIndex(tpl))
	r.Post("/form", handler.FormSubmit(tpl))
	r.Get("/healthz", handler.Health)
	r.NotFound(handler.NotFound(tpl))
}

/* ---------- Утилиты ---------- */

// generateNonce — генерирует случайный nonce для CSP
func generateNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
