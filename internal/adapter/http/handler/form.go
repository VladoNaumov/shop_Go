// ИСПРАВЛЕНО:
// - Рендерим НЕ "base", а "form" (обёртку страницы).

package handler

import (
	"net/http"

	"github.com/gorilla/csrf"
)

type FormViewModel struct {
	Title string
	OK    bool
}

func FormIndex(w http.ResponseWriter, r *http.Request) {
	vm := FormViewModel{
		Title: "Форма",
		OK:    r.URL.Query().Get("ok") == "1",
	}
	data := PageData{
		CSRFField: csrf.TemplateField(r),
		CSRFToken: csrf.Token(r),
		View:      vm,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.ExecuteTemplate(w, "form", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
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
