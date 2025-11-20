package view

import (
	"fmt"
	"html/template" // Стандартная библиотека Go для парсинга и рендеринга HTML-шаблонов (безопасно от XSS)
	"myApp/internal/core"

	"github.com/gin-gonic/gin"
	csrf "github.com/utrack/gin-csrf"
)

type Templates struct {
	templates map[string]*template.Template // Хранилище готовых шаблонов: ключ — имя страницы ("home"), значение — скомпилированный шаблон (layout + page)
}

type PageData struct {
	Title     string        // Заголовок страницы: используется в <title>{{.Title}}</title> в layout
	CSRFField template.HTML // Готовый HTML для скрытого CSRF-поля: <input type="hidden" name="csrf_token" value="..."> (template.HTML — чтобы не эскейпить HTML)
	Nonce     string        // CSP-nonce: случайная строка для защиты скриптов/стилей ({{.Nonce}} в шаблоне)
	Data      any           // Гибкие данные для страницы: struct, map и т.д. (передаётся в {{.Data}} в page-шаблоне)
}

const layoutFile = "web/templates/layouts/layout.html"

func New() (*Templates, error) {
	// Фиксированная map страниц — как в оригинале: ключи — имена для рендера ("home"), значения — пути к page-файлам
	// Почему map? Быстрый поиск по строке (O(1)). Легко добавлять/удалять страницы без сканирования FS.
	pages := map[string]string{
		"home":     "web/templates/pages/home.html",         // Главная страница
		"about":    "web/templates/pages/about.html",        // О проекте
		"form":     "web/templates/pages/form.html",         // Форма (с CSRF)
		"catalog":  "web/templates/pages/catalog.html",      // Каталог (список продуктов)
		"product":  "web/templates/pages/show_product.html", // Страница продукта (с data)
		"notfound": "web/templates/pages/404.html",          // 404-страница
	}

	// Шаг 1: Парсим layout ОДИН РАЗ (оптимизация!)
	// template.New("layout") — создаёт новый шаблон с именем "layout" (не влияет на рендер, только внутреннее).
	// ParseFiles(layoutFile) — читает файл, парсит в AST (абстрактное дерево), компилирует в исполняемый план.
	// Если layout сломан (синтаксис {{ }} неверный) — ошибка сразу.
	layoutTpl, err := template.New("layout").ParseFiles(layoutFile)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга layout: %w", err) // %w — "wrap" ошибки: сохраняет оригинальный стек для дебага
	}

	// Инициализируем структуру: пустая map для хранения готовых шаблонов (layout + page)
	t := &Templates{templates: make(map[string]*template.Template)}
	for name, pagePath := range pages {
		// Шаг 2: Клонируем layout и добавляем ТОЛЬКО page (не весь набор файлов)
		// layoutTpl.Clone() — создаёт независимую копию layout (чтобы добавление page не мутировало оригинал/другие страницы)
		// template.Must() — паникует при ошибке (но Clone редко фейлит; если да — приложение упадёт на старте)
		tpl := template.Must(layoutTpl.Clone()) // Клон — копия без мутации

		// ParseFiles(pagePath) — добавляет page-файл к клону: парсит {{define "base"}}...{{end}} из page и связывает с layout
		// _ — игнорируем возвращаемое имя шаблона (не нужно)
		_, err := tpl.ParseFiles(pagePath)
		if err != nil {
			return nil, fmt.Errorf("ошибка парсинга шаблона %q: %w", name, err) // Если page не существует или синтаксис сломан
		}

		// Проверяем, что шаблон "base" действительно существует (как в оригинале)
		// tpl.Lookup("base") — ищет скомпилированный блок по имени. Если page не имеет {{define "base"}} — ошибка.
		if tpl.Lookup("base") == nil {
			return nil, fmt.Errorf("в шаблонах отсутствует define \"base\" для страницы %s", name)
		}

		// Сохраняем готовый шаблон (layout + page) в map по имени страницы
		t.templates[name] = tpl
	}
	// Возвращаем готовый *Templates: все шаблоны в памяти, парсинг завершён.
	// Если ошибка — приложение не стартует (panic в main.go).
	return t, nil
}

// Render — метод структуры Templates: рендерит страницу в HTTP-ответ (c.Writer)
func (t *Templates) Render(c *gin.Context, templateName string, title string, data any) error {
	// Шаг 1: Ищем шаблон в map по имени (напр., "home")
	tpl, ok := t.templates[templateName]
	if !ok {
		// Если не найден — лог + ошибка (Gin вернёт 500 в роуте)
		core.LogError("Шаблон не найден", map[string]interface{}{"template": templateName})
		return fmt.Errorf("шаблон не найден: %s", templateName)
	}

	// Шаг 2: Извлекаем CSP-nonce из Gin-контекста (middleware добавил ранее, напр., c.Set(core.CtxNonce, "random123"))
	// Context().Value() — безопасно берёт значение по ключу (core.CtxNonce — твой константный ключ)
	nonceVal := c.Request.Context().Value(core.CtxNonce)
	nonce, _ := nonceVal.(string) // Приводим к string (если не string — паника, но middleware гарантирует)
	if nonce == "" {
		// Если nonce пуст — лог + ошибка (защита: без nonce CSP заблокирует скрипты)
		core.LogError("CSP Nonce не найден в контексте запроса", nil)
		return fmt.Errorf("nonce не найден")
	}

	// Шаг 3: Устанавливаем HTTP-заголовок для HTML (Gin по умолчанию text/plain, но для charset=utf-8 нужно явно)
	c.Header("Content-Type", "text/html; charset=utf-8")

	// Шаг 4: Генерируем CSRF-токен (utrack/gin-csrf хранит в сессии/cookie)
	// csrf.GetToken(c) — возвращает текущий токен для формы (если middleware CSRF включён)
	token := csrf.GetToken(c)
	csrfField := template.HTML("") // По умолчанию пусто (если нет форм — ок)
	if token != "" {
		// Формируем безопасный HTML: эскейпим только value (HTMLEscapeString), но весь input не эскейпим (template.HTML)
		// Почему вручную? utrack/gin-csrf не рендерит поле — только токен; мы добавляем <input>
		csrfField = template.HTML(fmt.Sprintf(`<input type="hidden" name="csrf_token" value="%s">`,
			template.HTMLEscapeString(token)))
	} else {
		// Если токен пуст (редко, если CSRF отключён) — лог, но продолжаем (не критично для GET-страниц)
		core.LogError("CSRF токен пуст", nil)
	}

	// Шаг 5: Собираем все данные в PageData — "модель" для шаблона (Go передаст как . в {{.Title}} и т.д.)
	page := PageData{
		Title:     title,     // Из вызова: "Главная страница"
		CSRFField: csrfField, // Готовый HTML для вставки в форму
		Nonce:     nonce,     // Для CSP в скриптах/стилях
		Data:      data,      // Твои данные: напр., gin.H{"Products": []Product{...}}
	}

	// Шаг 6: Рендерим! ExecuteTemplate(c.Writer, "base", page) — выполняет блок "base" из шаблона,
	// пишет HTML в HTTP-ответ (c.Writer — io.Writer). Layout + page сливаются динамически.
	// Если ошибка (редко: data невалидно) — возвращаем (роут обработает как 500).
	return tpl.ExecuteTemplate(c.Writer, "base", page)
}

//  Как это работает в нашей версии (Gin + utrack/gin-csrf):
//
// 1) При запуске сервера вызывается view.New() — шаблоны парсятся один раз и хранятся в памяти.
//    (В main.go: templates, _ := view.New(); r.SetTemplates(templates) или глобально)
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
  {{ .CSRFField }}  // Вставит CSRF-input автоматически
  <input type="text" name="name">
  <input type="email" name="email">
  <textarea name="message"></textarea>
  <button type="submit">Отправить</button>
</form>

В form.html: {{ define "base" }} ... {{ .CSRFField }} ... {{ end }}
*/
