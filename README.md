
***Проект интернет магазина ( Go 1.25.1 )***

---
## итоговая структура

```
myApp/
│
├─ cmd/
│  └─ app/
│     └─ main.go                 # запуск HTTP-сервера, graceful shutdown, CSRF, HSTS
│
├── internal/
│   ├── core/
│   │    ├── server.go       // Фабрика http.Server с таймаутами
│   │    ├── config.go       // Конфигурация приложения (AppName, Addr, Env, Secure, ...)
│   │    ├── router.go       //  маршрутизация (chi.Router)
│   │    ├── common.go       // Базовые middleware (лог, recover, timeout, CSP)
│   │    └── security.go     // Заголовки безопасности (CSP, XFO, MIME, Referrer)
│   └─ http/
│     └─ handler/
│        ├─ home.go              # главная страница
│        ├─ form.go              # форма + PRG-редирект
│        └─ about.go             # страница «О нас»
│
├─ web/
│  └─ templates/
│     ├─ layouts/base.gohtml     # {{define "base"}} ... {{block "content" .}}{{end}} ... {{end}}
│     ├─ partials/nav.gohtml     # {{define "nav"}} ... {{end}}
│     ├─ partials/footer.gohtml  # {{define "footer"}} ... {{end}}
│     └─ pages/
│        ├─ home.gohtml          # {{define "content"}} контент главной {{end}}
│        ├─ form.gohtml          # {{define "content"}} форма {{end}}
│        └─ about.gohtml         # {{define "content"}} о нас {{end}}
│
├─ make.bat                      # запуск, сборка, тесты, tidy; подхватывает .env
├─ go.mod                        # module awesomeProject
└─ go.sum
```

---

### **всё ядро веб-приложения**

```
myApp/
├─ cmd/
│  └─ app/
│     └─ main.go                 # запуск HTTP-сервера, graceful shutdown, CSRF, HSTS
├── internal/
│    └─ core/
│       ├── server.go       // Фабрика http.Server с таймаутами
│       ├── config.go       // Конфигурация приложения (AppName, Addr, Env, Secure, ...)
│       ├── router.go       // (пока не показан) — маршрутизация (chi.Router)
│       ├── common.go       // Базовые middleware (лог, recover, timeout, CSP)
│       └── security.go     // Заголовки безопасности (CSP, XFO, MIME, Referrer)
```

---

### 🔧 Что уже реализовано

| Компонент                    | Что делает                                                                         |
| ---------------------------- | ---------------------------------------------------------------------------------- |
| **`main.go`**                | Главная точка запуска. Настраивает логгер, конфиг, CSRF, HSTS и graceful shutdown. |
| **`config.go`**       | Все параметры приложения в одном месте. Можно запускать без `.env`.                |
| **`server.go`**          | Создаёт безопасный `http.Server` с таймаутами и базовой защитой от slow clients.   |
| **`common.go`**   | Подключает стандартные middleware (лог, IP, panic-recover, timeout, CSP).          |
| **`security.go`** | Добавляет заголовки безопасности (CSP, X-Frame-Options, Referrer-Policy и др.).    |

---

### 

✅  структура приложения
✅  безопасный старт сервера
✅  централизованные middleware
✅  защита (CSP, CSRF, HSTS, timeout, recover)
✅  читаемая и поддерживаемая архитектура

---

Следующий шаг — добавить **роутер и обработчики** (например, `/`, `/form`, `/api/...`),
чтобы сервер начал **отдавать страницы или JSON-ответы**.



---

## 🔹 Что уже сделано

✅ **Рабочие страницы:**

* `/` — главная
* `/about` — о компании
* `/form` (GET/POST) — форма с редиректом `303` после отправки

✅ **Простой запуск:**

```bash
make run      # загрузит .env и запустит go run ./cmd/app
make build    # соберёт bin\app.exe
make start    # запустит бинарь
make tidy     # обновит зависимости
make test     # прогоним тесты
```

---

## 🔹 Как добавить новую страницу

1️⃣ Создаёшь файл `web/templates/pages/contacts.gohtml`:

```html
{{ define "content" }}
<h1>Контакты</h1>
<p>Наш адрес: Hamina...</p>
{{ end }}
```

2️⃣ Создаёшь `internal/http/handler/contacts.go`:

```go
package handler
import (
  "html/template"
  "net/http"
)

func Contacts(w http.ResponseWriter, r *http.Request) {
  tpl := template.Must(template.ParseFiles(
    "web/templates/layouts/base.gohtml",
    "web/templates/partials/nav.gohtml",
    "web/templates/partials/footer.gohtml",
    "web/templates/pages/contacts.gohtml",
  ))
  w.Header().Set("Content-Type", "text/html; charset=utf-8")
  _ = tpl.ExecuteTemplate(w, "base", struct{ Title string }{"Контакты"})
}
```

3️⃣ Добавь маршрут в `router.go`:

```go
r.Get("/contacts", handler.Contacts)
```

# Как запускать

* Windows (батник): `make run`
  или: `go run ./cmd/app`
* Если удалял `go.mod`:
  `go mod init myApp && go mod tidy`




  
### 📄 `make.bat`

```bat
@echo off
if "%1"=="run" (
    echo 🔹 Running app...
    go run ./cmd/app
) else if "%1"=="build" (
    echo 🔹 Building binary...
    go build -o bin/app.exe ./cmd/app
) else if "%1"=="start" (
    echo 🔹 Starting binary...
    bin\app.exe
) else if "%1"=="clean" (
    echo 🔹 Cleaning build files...
    rmdir /s /q bin 2>nul
) else if "%1"=="test" (
    echo 🔹 Running Go tests...
    go test ./... -v
) else if "%1"=="lint" (
    echo 🔹 Running Go formatter...
    go fmt ./...
    echo 🔹 Running Go vet...
    go vet ./...
    echo ✅ Lint check completed.
) else (
    echo Usage: make [run^|build^|start^|clean^|test^|lint]
)
```

---

## ⚙️ Теперь доступно:

| Команда        | Описание                                           |
| -------------- | -------------------------------------------------- |
| `.\make run`   | запустить проект                                   |
| `.\make build` | собрать бинарник `bin\app.exe`                     |
| `.\make start` | запустить бинарник                                 |
| `.\make clean` | удалить `bin`                                      |
| `.\make test`  | запустить все Go-тесты                             |
| `.\make lint`  | форматировать и проверить код (`go fmt`, `go vet`) |

---


# Что уже сделано ✅

* **Структура проекта (Variant A, без embed)**: `cmd/app`, `internal/{app,config,http}`, `web/templates`.
* **Маршруты**: `/`, `/about`, `/form` (GET/POST с PRG).
* **Шаблоны разделены per-page**: каждый хендлер парсит **свой** `pages/<page>.gohtml` + общий `base/nav/footer`.
* **CSRF middleware**: подключён в `cmd/app/main.go` через `csrf.Protect(...)`.
* **Конфиг из env**: `internal/config/config.go` (поля: `AppName`, `Addr`, `Env`, `CSRFKey`, `Secure`).
* **Таймауты сервера**: `internal/app/server.go`.
* **HSTS в prod**: мидлварь `hsts` в `main.go` (включается только при `APP_ENV=prod`).
* **Запуск/сборка**: `make run`, `make build`, `make start`.


# Что осталось сделать (обновлённая дорожная карта) 🚧

1. **CSRF в шаблонах — довести до конца**

    * [ ] Вставлять токен в формы:

        * В `FormIndex` прокинуть в шаблон `{{ .CSRFField }}` (или просто вставить `csrf.TemplateField(r)`).
        * В `web/templates/pages/form.gohtml` добавить `{{ .CSRFField }}` внутри `<form>`.
    * Файлы: `internal/http/handler/form.go`, `web/templates/pages/form.gohtml`.

2. **Middleware + безопасность**

    * [ ] Добавить лёгкий набор:

        * `recover`, `timeout(15s)`, простой логгер запросов.
        * Security-заголовки: `Content-Security-Policy`, `X-Frame-Options`, `Referrer-Policy`, `X-Content-Type-Options`.
    * Файлы: `internal/http/middleware/{common.go,security.go}`, включить в `internal/http/router.go`.

3. **Конфиг — мелкие доработки**

    * [ ] Проверка наличия `CSRF_KEY` в prod (логировать предупреждение/фатал, если дефолт).
    * [ ] Вынести `HSTS` флаг в конфиг (на случай нестандартного прокси).
    * Файлы: `internal/config/config.go`, `cmd/app/main.go`.

4. **UX формы**

    * [ ] Отображать flash при `?ok=1` (зелёный alert на `/form`).
    * [ ] Базовая валидация: если `name`/`message` пустые — 400 + перерендер с текстом ошибки.
    * Файлы: `internal/http/handler/form.go`, `web/templates/pages/form.gohtml`.

5. **Статика**

    * [ ] Поднять `/assets/…`:

      ```go
      r.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.Dir("web/assets"))))
      ```
    * [ ] В prod добавить кэш-заголовки (`Cache-Control: public, max-age=31536000`) и хэши в именах файлов (по возможности).
    * Файлы: `internal/http/router.go`, `internal/http/middleware/security.go` (cache helper), `web/assets/...`.

6. **Тесты + качество**

    * [ ] Table-driven тесты хендлеров (`net/http/httptest`).
    * [ ] `go vet` уже есть в `make.bat`; позже подключить `golangci-lint`.
    * Файлы: `internal/http/handler/*_test.go`.

# Быстрые подсказки по внедрению

* **CSRF в форме** (минимум кода):

    * `FormIndex`:

      ```go
      data := struct{
        Title string
        CSRFField template.HTML
      }{"Форма", csrf.TemplateField(r)}
      ```
    * `form.gohtml` внутри `<form>`:

      ```html
      {{ .CSRFField }}
      ```

* **Security middleware** (очень простой вариант):

  ```go
  func SecureHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      w.Header().Set("X-Content-Type-Options", "nosniff")
      w.Header().Set("X-Frame-Options", "DENY")
      w.Header().Set("Referrer-Policy", "no-referrer-when-downgrade")
      // Базовый CSP; расширишь при необходимости
      w.Header().Set("Content-Security-Policy", "default-src 'self'")
      next.ServeHTTP(w, r)
    })
  }
  ```

компактный **план-график MVP** по спринтам с зависимостями, пакетами и критериями “готово”.

# Дорожная карта (таблица)

| Спринт | Цель                | Ключевые задачи                                                                                                     | Артефакты/критерии “готово”                                                      | Зависимости | Пакеты/инструменты                                      |
| ------ | ------------------- | ------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | ----------- | ------------------------------------------------------- |
| 1      | Каркас + SSR-база   | Структура проекта; chi + middleware; layout + `/`, `/catalog` (моки); статика; healthcheck; Makefile                | `go run` стартует; `/` и `/catalog` отдают HTML; `/healthz`=200; линтер проходит | —           | `chi`, `chi/middleware`, `golangci-lint`                |
| 2      | БД и миграции       | Docker Postgres; `migrations/0001_init.sql`; pgx; sqlc; интерфейсы `repo`                                           | `make migrate_up`; интеграционный тест `repo.Product.List()` зелёный             | 1           | `pgx/v5`, `golang-migrate`, `sqlc`, `testcontainers-go` |
| 3      | Каталог + товар     | Handlers `/catalog`, `/catalog/{sku}`; шаблоны; поиск/сортировка; seed данных                                       | Каталог рендерится из БД; карточка товара открывается; `httptest` на листинг     | 2           | stdlib `net/http/httptest`                              |
| 4      | Корзина + Checkout  | Сессии; `CartService`; `/cart`, `/checkout` (GET/POST); транзакция Order                                            | Полный путь: каталог → корзина → заказ → success; запись в БД                    | 3           | `scs/v2` (sessions)                                     |
| 5      | Админка             | `/admin/login`; AuthService (bcrypt/argon2id); CRUD товаров; CSRF                                                   | Админ логинится; создаёт/редактирует товар; CSRF включён; rate-limit логина      | 4           | `argon2id` или `bcrypt`, `nosurf`, `httprate`           |
| 6      | JSON API + OpenAPI  | `/api/v1/products`, `/api/v1/product/{sku}`, `/api/v1/cart`, `/api/v1/order`; единый формат ошибок; OpenAPI         | Контракты в `api/openapi.yaml`; контрактные тесты зелёные                        | 3–5         | `kin-openapi` + `oapi-codegen` (или swaggo)             |
| 7      | Продовая готовность | CSP/HSTS; TLS за reverse-proxy; `/metrics`; structured logs; Dockerfile/compose; CI (lint/test/build/image); бэкапы | HTTPS работает; Grafana видит метрики; образ публикуется; nightly backup         | 1–6         | `promhttp`, `slog`, Docker, Compose, GitHub Actions     |
| 8      | Полировка/UX/SEO    | Чистый UI; i18n ru/fi; favicon, sitemap, robots; мобильный UX                                                       | Lighthouse ok; базовая локализация; SEO-теги                                     | 1–7         | —                                                       |

---
