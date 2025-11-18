package view

import (
	"fmt"
	"html/template"

	"myApp/internal/core"

	"github.com/gin-gonic/gin"
	csrf "github.com/utrack/gin-csrf"
)

type Templates struct {
	templates map[string]*template.Template
}

type PageData struct {
	Title     string
	CSRFField template.HTML
	Nonce     string
	Data      any
}

const layoutFile = "web/templates/layouts/layout.html"

func New() (*Templates, error) {
	pages := map[string]string{
		"home":     "web/templates/pages/home.html",
		"about":    "web/templates/pages/about.html",
		"form":     "web/templates/pages/form.html",
		"catalog":  "web/templates/pages/catalog.html",
		"product":  "web/templates/pages/show_product.html",
		"notfound": "web/templates/pages/404.html",
	}

	t := &Templates{templates: make(map[string]*template.Template)}

	for name, pagePath := range pages {
		// Важно: сначала layout, потом страница — порядок имеет значение!
		files := []string{layoutFile, pagePath}

		tpl, err := template.New("").ParseFiles(files...)
		if err != nil {
			return nil, fmt.Errorf("ошибка парсинга шаблона %q: %w", name, err)
		}

		// Проверяем, что шаблон "base" действительно существует
		if tpl.Lookup("base") == nil {
			return nil, fmt.Errorf("в шаблонах отсутствует define \"base\" для страницы %s", name)
		}

		t.templates[name] = tpl
	}

	return t, nil
}

// Render — без изменений! Работает как раньше.
func (t *Templates) Render(c *gin.Context, templateName string, title string, data any) error {
	tpl, ok := t.templates[templateName]
	if !ok {
		core.LogError("Шаблон не найден", map[string]interface{}{"template": templateName})
		return fmt.Errorf("шаблон не найден: %s", templateName)
	}

	nonceVal := c.Request.Context().Value(core.CtxNonce)
	nonce, _ := nonceVal.(string)
	if nonce == "" {
		core.LogError("CSP Nonce не найден в контексте запроса", nil)
		return fmt.Errorf("nonce не найден")
	}

	c.Header("Content-Type", "text/html; charset=utf-8")

	token := csrf.GetToken(c)
	csrfField := template.HTML("")
	if token != "" {
		csrfField = template.HTML(fmt.Sprintf(`<input type="hidden" name="csrf_token" value="%s">`,
			template.HTMLEscapeString(token)))
	} else {
		core.LogError("CSRF токен пуст", nil)
	}

	page := PageData{
		Title:     title,
		CSRFField: csrfField,
		Nonce:     nonce,
		Data:      data,
	}

	return tpl.ExecuteTemplate(c.Writer, "base", page)
}

//  Как это работает в нашей версии (Gin + utrack/gin-csrf):
//
// 1) При запуске сервера вызывается view.New() — шаблоны парсятся один раз и хранятся в памяти.
//
// 2) Каждый Gin-хендлер вызывает tpl.Render(c, "имя", "заголовок", data).
//    Render получает nonce из Gin-контекста (кладётся middleware) и подготавливает PageData.
//
// 3) CSRF: токен берём из utrack/gin-csrf: token := csrf.GetToken(c).
//    Скрытое поле собираем вручную:
//       <input type="hidden" name="csrf_token" value="...">
//    (имя параметра — "csrf_token" по умолчанию у utrack/gin-csrf).
//
// 4) CSP: nonce пробрасывается в PageData.Nonce и используется в шаблоне:
//       <script nonce="{{ .Nonce }}">...</script>
//       <style  nonce="{{ .Nonce }}">...</style>
//    SecureHeaders() формирует CSP с разрешением по nonce.
//
// 5) Контент-тайп: Render ставит заголовок "Content-Type: text/html; charset=utf-8".
//
// 6) В base.html должен быть корневой шаблон с именем "base" ({{ define "base" }} ... {{ end }}),
//    в который дочерние страницы подключаются через {{ template }} или {{ block }}.

/*
<form method="POST" action="/form">
  {{ .CSRFField }}
  <input type="text" name="name">
  <input type="email" name="email">
  <textarea name="message"></textarea>
  <button type="submit">Отправить</button>
</form>

*/
