package handler

import (
	"html/template"
	"net/http"

	"github.com/gorilla/csrf"
)

func Home(w http.ResponseWriter, r *http.Request) {
	tpl := template.Must(template.ParseFiles(
		"web/templates/layouts/base.gohtml",
		"web/templates/partials/nav.gohtml",
		"web/templates/partials/footer.gohtml",
		"web/templates/pages/home.gohtml",
	))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tpl.ExecuteTemplate(w, "base",
		PageData{
			Title:     "Главная",
			CSRFField: csrf.TemplateField(r),
		})
}
