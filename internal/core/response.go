package core

// response.go — отвечает за единообразную структуру JSON-ответов,
// включая обработку ошибок по стандарту RFC 7807 ("Problem Details for HTTP APIs")

import (
	"net/http"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
)

// -----------------------------------------------------------
// ProblemDetail — стандартная структура ошибки по RFC 7807
// https://datatracker.ietf.org/doc/html/rfc7807
// -----------------------------------------------------------
//
// Поля описывают детальную ошибку API в едином формате:
//
//	{
//	  "type": "/errors/invalid_email",
//	  "title": "Bad Request",
//	  "status": 400,
//	  "detail": "Поле email указано неверно",
//	  "instance": "/errors/invalid_email",
//	  "code": "invalid_email",
//	  "fields": { "email": "неверный формат" }
//	}
//
// Клиенты (например, фронтенд) могут одинаково обрабатывать все ошибки.

type ProblemDetail struct {
	Type     string            `json:"type"`             // URI-идентификатор типа ошибки (у тебя /errors/{code})
	Title    string            `json:"title"`            // Короткое название (например "Bad Request")
	Status   int               `json:"status"`           // HTTP-статус (400, 404, 500 и т.д.)
	Detail   string            `json:"detail"`           // Подробное описание
	Instance string            `json:"instance"`         // URI конкретного случая (у тебя совпадает с Type)
	Code     string            `json:"code"`             // Внутренний код ошибки (например "invalid_email")
	Fields   map[string]string `json:"fields,omitempty"` // Ошибки по полям (для форм и валидации)
}

// -----------------------------------------------------------
// JSON — безопасная обёртка для c.JSON()
// -----------------------------------------------------------
// Просто отправляет любой объект как JSON-ответ с нужным статусом.
// Удобно для единообразия — не писать каждый раз c.JSON().

func JSON(c *gin.Context, status int, v any) {
	c.JSON(status, v)
}

// -----------------------------------------------------------
// FailC — централизованная обработка ошибок API
// -----------------------------------------------------------
// Используется во всех местах, где нужно:
//  1. Преобразовать ошибку в структурированную форму (через From(err))
//  2. Прологировать ошибку
//  3. Вернуть клиенту JSON-ответ в формате ProblemDetail
//  4. Завершить цепочку middleware (Abort)

func FailC(c *gin.Context, err error) {
	// ae (AppError) — твоя обёртка для внутренних ошибок (core/app_error.go)
	// В ней есть код, HTTP-статус, сообщение и т.д.
	ae := From(err)

	// Получаем уникальный идентификатор запроса (request_id)
	reqID := requestid.Get(c)
	if reqID == "" {
		// Если middleware requestid не сработал, пробуем из заголовка
		reqID = c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = "n/a"
		}
	}

	// Логируем ошибку в единообразном виде
	LogError("Ошибка запроса", map[string]interface{}{
		"request_id": reqID,        // ID запроса (для трейсинга)
		"path":       c.FullPath(), // URL маршрута (например /api/users/:id)
		"code":       ae.Code,      // Внутренний код ошибки (например "db_error")
		"status":     ae.Status,    // HTTP статус
		"message":    ae.Message,   // Сообщение для пользователя
		"fields":     ae.Fields,    // Ошибки по полям (если есть)
		"error":      ae.Err,       // Исходная ошибка (Go error)
	})

	// Готовим тело ответа в формате RFC 7807
	problem := ProblemDetail{
		Type:     "/errors/" + ae.Code,       // URI типа ошибки
		Title:    http.StatusText(ae.Status), // Название по статусу (например "Not Found")
		Status:   ae.Status,                  // Код HTTP
		Detail:   ae.Message,                 // Детальное сообщение
		Instance: "/errors/" + ae.Code,       // Совпадает с type (но может быть разным)
		Code:     ae.Code,                    // Твой внутренний код
		Fields:   ae.Fields,                  // Ошибки по полям (если есть)
	}

	// Отправляем JSON-клиенту
	JSON(c, ae.Status, problem)

	// Прерываем дальнейшие middleware/обработчики
	c.Abort()
}

/*

### 🧩 Что происходит шаг за шагом в `FailC`

| Шаг | Действие                               | Пример                               |
| --- | -------------------------------------- | ------------------------------------ |
| 1   | Получаем `ae := From(err)`             | Конвертирует `error` → `AppError`    |
| 2   | Извлекаем `request_id`                 | Позволяет связать логи и запрос      |
| 3   | Логируем всё через `LogError()`        | Записывает в файл/консоль            |
| 4   | Формируем структуру `ProblemDetail`    | Готовим JSON для ответа              |
| 5   | Отправляем JSON с нужным HTTP-статусом | Например, 400 Bad Request            |
| 6   | `c.Abort()` останавливает цепочку      | Gin не вызывает следующие middleware |

---

💡 **Зачем всё это:**
Ты получаешь **единый формат ошибок** на всём API —
удобно для фронтенда, логирования, и тестов (Postman, Jest и т.д.).

---


*/
