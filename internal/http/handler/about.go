package handler

// about.go
import (
	"net/http"

	"myApp/internal/core"
	"myApp/internal/view"

	"github.com/gin-gonic/gin"
)

// Структура для данных, которые будут переданы в шаблон
type AboutData struct {
	PageTitle string
	Content   string
	Year      int
}

// About — обработчик страницы "О нас" (OWASP A03: Injection)
func About(tpl *view.Templates) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Создаем данные для передачи
		data := AboutData{
			PageTitle: "О нашей компании",
			Content:   "Мы — команда профессионалов, создающих отличные приложения.",
			Year:      2025, // Пример динамического значения
		}

		// Передаем структуру data в качестве последнего аргумента
		if err := tpl.Render(c, "about", "О нас", data); err != nil {
			core.LogError("Ошибка рендеринга шаблона about", map[string]interface{}{
				"error": err.Error(),
				"path":  c.Request.URL.Path,
			})
			c.String(http.StatusInternalServerError, "Ошибка отображения страницы")
			return
		}
	}
}
