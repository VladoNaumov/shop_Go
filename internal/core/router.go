package core

// route.go

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"myApp/internal/http/handler"
)

// NewRouter создаёт chi-маршрутизатор и подключает все middleware.
// Здесь же определяются основные маршруты (страницы, формы, API, статика).
func NewRouter() http.Handler {
	r := chi.NewRouter()

	// --- Общие middleware ---
	UseCommon(r)

	// --- Основные страницы ---
	r.Get("/", handler.Home)
	r.Get("/about", handler.About)

	// --- Форма (GET/POST) ---
	r.Get("/form", handler.FormIndex)
	r.Post("/form", handler.FormSubmit)

	// --- Статические файлы ---
	// Отдаём всё из каталога web/assets по пути web/assets/*
	static := http.FileServer(http.Dir("web/assets"))

	// [USED HERE] — подключаем cacheStatic только в prod
	if os.Getenv("APP_ENV") == "prod" {
		static = cacheStatic(static)
	}

	r.Handle("/assets/*",
		http.StripPrefix("/assets/", static),
	)

	// --- Healthcheck (для мониторинга) ---
	r.Get("/healthz", handler.Health)

	// --- Обработка 404 ---
	r.NotFound(handler.NotFound)

	return r
}

// cacheStatic — добавляет долгоживущий кэш для статики.
// В dev (APP_ENV != "prod") заголовок не ставим — чтобы было удобно разрабатывать.
func cacheStatic(next http.Handler) http.Handler { // [ADDED]
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Добавляем заголовки кэширования для продакшена
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("Vary", "Accept-Encoding")
		next.ServeHTTP(w, r)
	})
}
