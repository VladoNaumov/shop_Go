package handlers

// SSR для главной: Route -> (этот "контроллер") -> Template.
// Шаблоны встраиваем через embed, чтобы не ловить проблемы glob/слешей на Windows.

import (
	"html/template"
	"net/http"
)

// Встраиваем шаблоны по папкам (без **).
// Относительные пути заданы от текущего файла.
var (
	tpl = template.Must(template.ParseFiles(
		"web/templates/layouts/base.gohtml",
		"web/templates/partials/nav.gohtml",
		"web/templates/partials/footer.gohtml",
		"web/templates/pages/home.gohtml",
	))
)

// ViewModel для страницы
type HomeViewsModel struct {
	Title   string
	Message string
}

// Хендлер главной страницы (аналог HomeController@index)
func HomeIndex(w http.ResponseWriter, r *http.Request) {
	vm := HomeViewsModel{
		Title:   "Главная",
		Message: "Это стартовая страница. SSR на html/template + chi.",
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Рендерим layout "base"; внутри он вставит блок {{block "content"}} из pages/home.tmpl
	if err := tpl.ExecuteTemplate(w, "base", vm); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}
