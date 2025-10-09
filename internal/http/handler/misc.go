package handler

import (
	"encoding/json"
	"html/template"
	"net/http"
)

// Health — простой healthcheck без зависимости от core.
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// NotFound — страница 404.
func NotFound(w http.ResponseWriter, r *http.Request) {
	tpl := template.Must(template.ParseFiles(
		"web/templates/layouts/base.gohtml",
		"web/templates/partials/nav.gohtml",
		"web/templates/partials/footer.gohtml",
		"web/templates/pages/404.gohtml",
	))
	w.WriteHeader(http.StatusNotFound)
	_ = tpl.ExecuteTemplate(w, "base", struct{ Title string }{"Страница не найдена"})
}
