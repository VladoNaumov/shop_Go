package handler

import (
	"html/template"
	"net/http"
)

func FormIndex(w http.ResponseWriter, r *http.Request) {
	tpl := template.Must(template.ParseFiles(
		"web/templates/layouts/base.gohtml",
		"web/templates/partials/nav.gohtml",
		"web/templates/partials/footer.gohtml",
		"web/templates/pages/form.gohtml",
	))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tpl.ExecuteTemplate(w, "base", PageData{Title: "Форма"})
}

func FormSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/form?ok=1", http.StatusSeeOther)
}
