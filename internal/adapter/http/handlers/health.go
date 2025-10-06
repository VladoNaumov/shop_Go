package handlers

// Простые JSON-эндпоинты для liveness/readiness.
// SSR для главной: Route -> (этот "контроллер") -> json

import (
	"encoding/json"
	"net/http"
)

func Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func Ready(w http.ResponseWriter, r *http.Request) {
	// Здесь позже: проверки БД/кэша. Сейчас — "готов".
	writeJSON(w, http.StatusOK, map[string]any{"ready": true})
}

// Вспомогательная функция для единообразных JSON-ответов
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
