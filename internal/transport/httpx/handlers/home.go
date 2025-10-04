package handlers

// HTML-обработчик:
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

// В неё кладутся данные, которые потом будут вставлены в HTML-шаблон (.tmpl).
// 💡 То есть это как «контейнер с переменными для шаблона».
type HomeViewsModel struct {
	Title   string
	Message string
}

// Хендлер главной страницы (аналог HomeController@index)
// http.ResponseWriter — куда писать ответ (HTML, JSON, текст и т.д.);
func HomeIndex(w http.ResponseWriter, r *http.Request) {

	// Создание данных для шаблона
	vm := HomeViewsModel{
		Title:   "Главная",
		Message: "Это стартовая страница. SSR на html/template + chi.",
	}
	// Установка типа контента
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Рендерим layout "base"; внутри он вставит блок {{block "content"}} из pages/home.tmpl
	if err := tpl.ExecuteTemplate(w, "base", vm); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}
