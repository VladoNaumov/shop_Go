package handlers

import (
	"html/template"
	"net/http"
)

var tpl = template.Must(template.ParseFiles(
	"web/templates/layouts/base.gohtml",
	"web/templates/partials/nav.gohtml",
	"web/templates/partials/footer.gohtml",
	"web/templates/pages/home.gohtml",
))

type HomeVM struct {
	Title   string
	Message string
}

func Home(w http.ResponseWriter, r *http.Request) {
	vm := HomeVM{
		Title:   "Главная",
		Message: "Это стартовая страница. SSR на html/template + chi.",
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.ExecuteTemplate(w, "base", vm); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}
