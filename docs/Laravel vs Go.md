### 1️⃣ Маршрут (Route)

В Laravel:

```php
Route::get('/', [HomeController::class, 'index']);
```

В Go (через chi):

```go
r.Get("/", handlers.Home)
```

То есть `handlers.Home` — это **функция-обработчик (handler)**, которую chi вызывает при обращении к `/`.

---

### 2️⃣ Хендлер (аналог контроллера)

В Laravel метод контроллера принимает Request и возвращает View:

```php
public function index() {
    return view('home', ['title' => 'Главная']);
}
```

А в Go хендлер — это просто функция с сигнатурой:

```go
func(w http.ResponseWriter, r *http.Request)
```

Она **сама** отвечает пользователю (в отличие от Laravel, где фреймворк делает это за тебя).
Вот как это делается:

```go
func Home(w http.ResponseWriter, r *http.Request) {
    vm := HomeVM{ // ViewModel
        Title:   "Главная",
        Message: "Это стартовая страница.",
    }
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    if err := tpl.ExecuteTemplate(w, "base", vm); err != nil {
        http.Error(w, "template error", http.StatusInternalServerError)
    }
}
```

Здесь `tpl.ExecuteTemplate` — это аналог Laravel’евского `return view()`.

---

### 3️⃣ Подключение шаблонов (`tpl`)

Перед этим в коде мы создаём переменную `tpl`, которая хранит **скомпилированные шаблоны**:

```go
var tpl = template.Must(template.ParseGlob("web/templates/**/*.tmpl"))
```

Это значит:

* Go загружает и компилирует все `.tmpl` файлы (layouts, partials, pages).
* Хранит их в памяти.
* Потом `tpl.ExecuteTemplate()` выбирает из них конкретный шаблон (например, `"base"`), подставляет данные и выводит HTML в `w`.

---

### 4️⃣ ViewModel (данные для шаблона)

В Go нет `array` как в PHP-контроллере, но мы делаем структурку:

```go
type HomeVM struct {
    Title   string
    Message string
}
```

И передаём её в шаблон:

```go
tpl.ExecuteTemplate(w, "base", vm)
```

Внутри `base.tmpl` и `home.tmpl` можно использовать `{{.Title}}` и `{{.Message}}`.

---

### 🔗 Вся цепочка:

| Laravel                          | Go (chi + html/template)                  |
| -------------------------------- | ----------------------------------------- |
| Route → Controller → View        | Route → Handler → Template                |
| Контроллер возвращает `view()`   | Handler вызывает `tpl.ExecuteTemplate()`  |
| View получает массив переменных  | Template получает структуру (ViewModel)   |
| Response формируется фреймворком | Handler сам пишет HTML в `ResponseWriter` |

---