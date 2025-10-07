package handler

import (
	"net/http"

	"github.com/gorilla/csrf"
)

type AboutVM struct {
	Title string
}

func About(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		CSRFField: csrf.TemplateField(r),
		CSRFToken: csrf.Token(r),
		View:      AboutVM{Title: "О нас"},
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.ExecuteTemplate(w, "about", data); err != nil { // <-- рендерим "about", не "base"
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}
