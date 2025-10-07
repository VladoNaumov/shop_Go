package handler

import (
	"html/template"
	"net/http"
)

var aboutTpl = template.Must(template.ParseFiles(
	"web/templates/layouts/base.gohtml",
	"web/templates/partials/nav.gohtml",
	"web/templates/partials/footer.gohtml",
	"web/templates/pages/about.gohtml",
))

func About(w http.ResponseWriter, r *http.Request) {
	err := aboutTpl.ExecuteTemplate(w, "base", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
