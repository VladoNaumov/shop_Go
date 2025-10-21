package core

//response.go
import (
	"net/http"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
)

// ProblemDetail — тот же RFC7807
type ProblemDetail struct {
	Type     string            `json:"type"`
	Title    string            `json:"title"`
	Status   int               `json:"status"`
	Detail   string            `json:"detail"`
	Instance string            `json:"instance"`
	Code     string            `json:"code"`
	Fields   map[string]string `json:"fields,omitempty"`
}

// JSON — отправка JSON через Gin (аналог вашей JSON)
func JSON(c *gin.Context, status int, v any) {
	// Content-Type и кодировка расставятся автоматически,
	// но явно — безопаснее и нагляднее.
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.JSON(status, v)
}

// Fail — ответ об ошибке в формате RFC7807 + логирование
func FailC(c *gin.Context, err error) { // новое имя, чтобы не конфликтовать с вашей версией
	ae := From(err)

	// Пытаемся достать request_id из middleware gin-contrib/requestid.
	reqID := requestid.Get(c)
	if reqID == "" {
		// Фолбэк — из заголовка, если внешний балансер/прокси его дал.
		reqID = c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = "n/a"
		}
	}

	logFields := map[string]interface{}{
		"request_id": reqID,
		"path":       c.FullPath(),
		"code":       ae.Code,
		"status":     ae.Status,
		"message":    ae.Message,
		"fields":     ae.Fields,
		"error":      ae.Err,
	}
	LogError("Ошибка обработки запроса", logFields)

	problem := ProblemDetail{
		Type:     "/errors/" + ae.Code,
		Title:    http.StatusText(ae.Status),
		Status:   ae.Status,
		Detail:   ae.Message,
		Instance: c.Request.URL.Path, // фактический URL
		Code:     ae.Code,
		Fields:   ae.Fields,
	}
	JSON(c, ae.Status, problem)
	c.Abort()
}
