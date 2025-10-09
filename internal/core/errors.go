package core

//errors.go
import (
	"errors"
	"fmt"
	"net/http"
)

// AppError — единый контейнер для ошибок приложения.
type AppError struct {
	Code    string            // машинный код ("validation", "not_found", "internal" и т.п.)
	Status  int               // HTTP-статус
	Message string            // безопасное для клиента сообщение
	Err     error             // первопричина (не уходит клиенту)
	Fields  map[string]string // поле -> текст ошибки (для валидации форм/API)
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s (%d): %s: %v", e.Code, e.Status, e.Message, e.Err)
	}
	return fmt.Sprintf("%s (%d): %s", e.Code, e.Status, e.Message)
}
func (e *AppError) Unwrap() error { return e.Err }

// Фабрики

func BadRequest(msg string, fields map[string]string) *AppError {
	return &AppError{Code: "bad_request", Status: http.StatusBadRequest, Message: msg, Fields: fields}
}
func NotFound(msg string) *AppError {
	return &AppError{Code: "not_found", Status: http.StatusNotFound, Message: msg}
}
func Forbidden(msg string) *AppError {
	return &AppError{Code: "forbidden", Status: http.StatusForbidden, Message: msg}
}
func Unauthorized(msg string) *AppError {
	return &AppError{Code: "unauthorized", Status: http.StatusUnauthorized, Message: msg}
}
func Internal(msg string, err error) *AppError {
	return &AppError{Code: "internal", Status: http.StatusInternalServerError, Message: msg, Err: err}
}

// From — приводит произвольную ошибку к *AppError* (по умолчанию Internal 500).
func From(err error) *AppError {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae
	}
	if err == nil {
		return nil
	}
	return Internal("internal error", err)
}
