package core

import (
	"net/http"

	"myApp/internal/http/handler" // пакет с обработчиками страниц

	"github.com/go-chi/chi/v5"
)

// NewRouter создаёт chi-маршрутизатор и подключает все middleware.
// Здесь же определяются основные маршруты (страницы, формы, API, статика).
func NewRouter() http.Handler {
	r := chi.NewRouter()

	// --- Общие middleware ---
	// Подключаем RequestID, Logger, Recover, Timeout, CSP и др.
	UseCommon(r)

	// --- Основные страницы ---
	r.Get("/", handler.Home)       // главная страница
	r.Get("/about", handler.About) // страница "О нас"

	// --- Форма (GET/POST) ---
	r.Get("/form", handler.FormIndex)   // показать форму
	r.Post("/form", handler.FormSubmit) // обработать отправку формы

	// --- Статические файлы ---
	// Отдаём всё из каталога web/assets по пути /assets/*
	// Например: /assets/css/style.css → ./web/assets/css/style.css
	r.Handle("/assets/*",
		http.StripPrefix("/assets/",
			http.FileServer(http.Dir("web/assets")),
		),
	)

	// --- Healthcheck (для мониторинга) ---
	r.Get("/healthz", handler.Health) // возвращает 200 OK, если сервер жив

	// --- Обработка 404 ---
	r.NotFound(handler.NotFound) // если маршрут не найден

	return r
}
