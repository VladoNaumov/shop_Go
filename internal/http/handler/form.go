package handler

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/csrf"
)

type PageData struct {
	Title     string
	CSRFField template.HTML
	OK        bool
}

type FormData struct {
	Name    string `validate:"required,min=2,max=100"`
	Email   string `validate:"required,email"`
	Message string `validate:"required,max=2000"`
}

type FormView struct {
	PageData
	Form   FormData
	Errors map[string]string
}

// [ADDED] Инициализация валидатора один раз на пакет
var validate = validator.New()

// ---------------------------------------

func FormIndex(w http.ResponseWriter, r *http.Request) {
	tpl := template.Must(template.ParseFiles(
		"web/templates/layouts/base.gohtml",
		"web/templates/partials/nav.gohtml",
		"web/templates/partials/footer.gohtml",
		"web/templates/pages/form.gohtml",
	))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	ok := r.URL.Query().Get("ok") == "1"

	data := FormView{
		PageData: PageData{
			Title:     "Форма",
			CSRFField: csrf.TemplateField(r),
			OK:        ok,
		},
		Form:   FormData{},
		Errors: map[string]string{},
	}
	_ = tpl.ExecuteTemplate(w, "base", data)
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

	// Чтение и нормализация данных
	f := FormData{
		Name:    strings.TrimSpace(r.Form.Get("name")),
		Email:   strings.TrimSpace(r.Form.Get("email")),
		Message: strings.TrimSpace(r.Form.Get("message")),
	}

	// Валидация через validator/v10
	errs := map[string]string{}
	if err := validate.Struct(f); err != nil {
		if verrs, ok := err.(validator.ValidationErrors); ok {
			for _, e := range verrs {
				switch e.Field() {
				case "Name":
					switch e.Tag() {
					case "required":
						errs["name"] = "Укажите имя"
					case "min":
						errs["name"] = "Имя должно быть не короче 2 символов"
					case "max":
						errs["name"] = "Слишком длинное имя (макс. 100)"
					default:
						errs["name"] = "Некорректное имя"
					}
				case "Email":
					switch e.Tag() {
					case "required":
						errs["email"] = "Укажите email"
					case "email":
						errs["email"] = "Введите корректный email"
					default:
						errs["email"] = "Некорректный email"
					}
				case "Message":
					switch e.Tag() {
					case "required":
						errs["message"] = "Напишите сообщение"
					case "max":
						errs["message"] = "Слишком длинное сообщение (макс. 2000)"
					default:
						errs["message"] = "Некорректное сообщение"
					}
				}
			}
		} else {
			// Общий случай ошибки
			errs["form"] = "Ошибка валидации"
		}
	}

	// Если ошибки — ререндер формы с Bootstrap-классами и текстами ошибок
	if len(errs) > 0 {
		tpl := template.Must(template.ParseFiles(
			"web/templates/layouts/base.gohtml",
			"web/templates/partials/nav.gohtml",
			"web/templates/partials/footer.gohtml",
			"web/templates/pages/form.gohtml",
		))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = tpl.ExecuteTemplate(w, "base", FormView{
			PageData: PageData{
				Title:     "Форма",
				CSRFField: csrf.TemplateField(r),
			},
			Form:   f,
			Errors: errs,
		})
		return
	}

	// [NOTE] Здесь можно отправить email/положить в очередь и т.п.
	// [REMOVED] Любые записи в БД о факте отправки — по твоей просьбе НЕ выполняем.

	// [ADDED] PRG: редирект на GET с флагом ok=1
	http.Redirect(w, r, "/form?ok=1", http.StatusSeeOther)
}
