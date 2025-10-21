package core

// response.go
import (
	"net/http"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
)

// ProblemDetail — RFC 7807
type ProblemDetail struct {
	Type     string            `json:"type"`
	Title    string            `json:"title"`
	Status   int               `json:"status"`
	Detail   string            `json:"detail"`
	Instance string            `json:"instance"`
	Code     string            `json:"code"`
	Fields   map[string]string `json:"fields,omitempty"`
}

// JSON — просто отправка
func JSON(c *gin.Context, status int, v any) {
	c.JSON(status, v)
}

// FailC — ошибка с логированием
func FailC(c *gin.Context, err error) {
	ae := From(err)

	reqID := requestid.Get(c)
	if reqID == "" {
		reqID = c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = "n/a"
		}
	}

	LogError("Ошибка запроса", map[string]interface{}{
		"request_id": reqID,
		"path":       c.FullPath(),
		"code":       ae.Code,
		"status":     ae.Status,
		"message":    ae.Message,
		"fields":     ae.Fields,
		"error":      ae.Err,
	})

	problem := ProblemDetail{
		Type:     "/errors/" + ae.Code,
		Title:    http.StatusText(ae.Status),
		Status:   ae.Status,
		Detail:   ae.Message,
		Instance: "/errors/" + ae.Code, // ← ИСПРАВЛЕНО: URI ошибки, не путь запроса
		Code:     ae.Code,
		Fields:   ae.Fields,
	}

	JSON(c, ae.Status, problem)
	c.Abort()
}
