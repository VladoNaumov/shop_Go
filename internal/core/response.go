package core

// response.go — стандартные функции для JSON-ответов и обработки ошибок API.

import (
	"encoding/json"
	"log"
	"net/http"
)

// JSON — стандартный JSON-ответ.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// NoContent — отправляет HTTP 204 без тела.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// ProblemDetail — структура в формате RFC7807 (ошибки API).
type ProblemDetail struct {
	Type     string            `json:"type,omitempty"`
	Title    string            `json:"title"`
	Status   int               `json:"status"`
	Detail   string            `json:"detail,omitempty"`
	Instance string            `json:"instance,omitempty"`
	Code     string            `json:"code,omitempty"`
	Fields   map[string]string `json:"fields,omitempty"`
}

// Fail — единая точка для обработки ошибок API.
// Преобразует error → AppError → JSON-ответ с деталями.
func Fail(w http.ResponseWriter, r *http.Request, err error) {
	ae := From(err)

	// Пишем ошибку в стандартный лог (в консоль)
	log.Printf("ERROR: code=%s status=%d message=%s fields=%v err=%v",
		ae.Code, ae.Status, ae.Message, ae.Fields, ae.Err)

	// Отправляем JSON с деталями ошибки
	JSON(w, ae.Status, ProblemDetail{
		Title:  http.StatusText(ae.Status),
		Status: ae.Status,
		Detail: ae.Message,
		Code:   ae.Code,
		Fields: ae.Fields,
	})
}
