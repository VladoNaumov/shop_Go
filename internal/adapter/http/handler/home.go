// ИСПРАВЛЕНО:
// - Рендерим НЕ "base", а "home" (обёртку страницы).
// - Удалён второй лишний ExecuteTemplate.

package handler

import (
	"html/template"
	"net/http"

	"github.com/gorilla/csrf"
)

var tpl = template.Must(template.ParseFiles(
	"web/templates/layouts/base.gohtml",
	"web/templates/partials/nav.gohtml",
	"web/templates/partials/footer.gohtml",
	"web/templates/pages/home.gohtml",
	"web/templates/pages/form.gohtml",
	"web/templates/pages/about.gohtml",
))

type HomeViewsModel struct {
	Title   string
	Message string
}

type PageData struct {
	CSRFField template.HTML
	CSRFToken string
	View      any
}

func HomeIndex(w http.ResponseWriter, r *http.Request) {
	vm := HomeViewsModel{
		Title:   "Главная",
		Message: "Это стартовая страница. SSR на html/template + chi.",
	}
	data := PageData{
		CSRFField: csrf.TemplateField(r),
		CSRFToken: csrf.Token(r),
		View:      vm,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.ExecuteTemplate(w, "home", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}
