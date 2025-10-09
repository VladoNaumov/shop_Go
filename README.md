
***Проект интернет магазина ( Go 1.25.1 )***

## Task 1. 

### **ядро WEB server**

```
myApp/
├─ cmd/
│  └─ app/
│     └─ main.go                 # запуск HTTP-сервера, graceful shutdown, CSRF, HSTS
├── internal/
│    └─ core/
│       ├── server.go       // Фабрика http.Server с таймаутами
│       ├── config.go       // Конфигурация приложения (AppName, Addr, Env, Secure, ...)
│       ├── router.go       // маршрутизация (chi.Router)
│       ├── common.go       // Базовые middleware (лог, recover, timeout, CSP)
│       ├── security.go     // Заголовки безопасности (CSP, XFO, MIME, Referrer)
│       ├── logger.go       // логирование (не реализовано)
│       ├── errors.go       // унифицированные ошибки (не реализовано)
│       └── response.go     // стандартные ответы (не реализовано)
```

---

### 🔧 Что уже реализовано

| Компонент                | Что делает                                                                         |
|--------------------------| ---------------------------------------------------------------------------------- |
| **`main.go`**            | Главная точка запуска. Настраивает логгер, конфиг, CSRF, HSTS и graceful shutdown. |
| **`config.go`**          | Все параметры приложения в одном месте. Можно запускать без `.env`.                |
| **`server.go`**          | Создаёт безопасный `http.Server` с таймаутами и базовой защитой от slow clients.   |
| **`common.go`**          | Подключает стандартные middleware (лог, IP, panic-recover, timeout, CSP).          |
| **`security.go`**        | Добавляет заголовки безопасности (CSP, X-Frame-Options, Referrer-Policy и др.).    |

---


Вот полный список — **всё, что обычно входит в ядро (`/internal/core`)** современного Go-приложения:

---

### 🧩 Обязательные файлы ядра

1. `config.go` — конфигурация приложения
2. `server.go` — создание и настройка HTTP-сервера
3. `router.go` — инициализация и регистрация маршрутов
4. `common.go` — подключение общих middleware
5. `security.go` — заголовки безопасности
6. `logger.go` — настройка глобального логгера
7. `errors.go` — унифицированная обработка ошибок
8. `response.go` — стандартные функции для HTTP-ответов (JSON, error, redirect)

---

### ⚙️ Дополнительные (рекомендуемые) файлы ядра

9. `context.go` — функции для работы с `context.Context` (userID, traceID и т.п.)
10. `utils.go` — вспомогательные утилиты (хэш, UUID, форматирование, время)
11. `graceful.go` — отдельная реализация graceful shutdown (если не в main)
12. `env.go` — чтение переменных окружения и их валидация
13. `metrics.go` — метрики (Prometheus, health-check)
14. `buildinfo.go` — информация о сборке (версия, commit, дата)

---

📘
Твои текущие файлы:
`config.go`, `server.go`, `common.go`, `security.go`, `router.go` — ✅ уже готовы.
Осталось добавить:
➡️ `logger.go`, `errors.go`, `response.go` — чтобы ядро было **полностью завершено и самодостаточно**.

---

## Task 2.

добавить **роутер и обработчики** (например, `/`, `/form`, `/about`, `/api/...`),
чтобы сервер начал **отдавать страницы или JSON-ответы**.


---

✅ **Рабочие страницы:**

* `/` — главная
* `/about` — о компании
* `/form` (GET/POST) — форма с редиректом `303` после отправки

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
    "web/templates/pages/contacts.gohtml",  // <--- new
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
* Если удалял `go.mod`: `go mod init myApp && go mod tidy`



# Что уже сделано ✅

* **Структура проекта (без embed)**: `cmd/app`, `internal/{app,config,http}`, `web/templates`.
* **Маршруты**: `/`, `/about`, `/form` (GET/POST с PRG).
* **Шаблоны разделены per-page**: каждый хендлер парсит **свой** `pages/<page>.gohtml` + общий `base/nav/footer`.
* **CSRF middleware**: подключён в `cmd/app/main.go` через `csrf.Protect(...)`.
* **Конфиг** config.go
* **Таймауты сервера**: `internal/app/server.go`.
* **HSTS в prod**: мидлварь `hsts` в `main.go` (включается только при `APP_ENV=prod`).
* **Запуск/сборка**: `make run`, `make build`, `make start`.


--

## 🧩 **Текущее состояние проекта**

| №     | Раздел                           | Статус            | Комментарий                                                                                                                                                               |
| ----- | -------------------------------- | ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **1** | **CSRF в шаблонах**              | ✅ **Сделано**     | CSRF-токен вставляется через `csrf.TemplateField(r)` и доступен в `{{ .CSRFField }}`. Все формы защищены (`FormIndex`, `FormSubmit`, шаблон `form.gohtml`).               |
| **2** | **Middleware + безопасность**    | ✅ **Сделано**     | Подключены `recover`, `timeout(15s)`, `logger`, `CSP`, `X-Frame-Options`, `Referrer-Policy`, `X-Content-Type-Options`. Всё в `core/common.go` и `core/secure_headers.go`. |
| **3** | **Конфиг (CSRF_KEY, HSTS)**      | ✅ **Сделано**     | Конфигурация теперь читается из окружения (`core/config.go`), есть проверка дефолтного ключа в prod, `HSTS` включается при `Secure=true`.                                 |
| **4** | **UX формы (валидация + flash)** | ✅ **Сделано**     | Используется `go-playground/validator/v10`. Серверная валидация, ререндеринг ошибок, flash при `?ok=1`. Всё реализовано.                                                  |
| **5** | **Статика / Cache-Control**      | ✅ **Сделано**     | `cacheStatic()` добавляет `Cache-Control: public, max-age=31536000, immutable` и `Vary: Accept-Encoding` только в prod. Подключено в `core/router.go`.                    |
| **6** | **Тесты + качество**             | 🚧 **В процессе** | Пока нет `httptest` для `FormSubmit` и `FormIndex`, и не добавлен `golangci-lint`. План на следующий шаг.                                                                 |

---

## 🚀 **Что осталось сделать (финальные задачи)**

### 1. **Тесты**

* Написать `httptest` для `FormIndex` и `FormSubmit`:

    * Проверка статуса 200/303/400
    * Проверка, что CSRF-токен есть в HTML
    * Проверка обработки ошибок формы
* Добавить unit-тест для `cacheStatic()` (заголовки кэша присутствуют)
* Использовать `testing.T.Run` для table-driven тестов

### 2. **Линтеры и статический анализ**

* Добавить `golangci-lint` в make-скрипт или GitHub Actions.
  Проверки: `govet`, `gofmt`, `gocyclo`, `staticcheck`, `errcheck`.

### 3. **Dev / Prod UX**

* Добавить `.env.example` с основными переменными (`APP_ENV`, `CSRF_KEY`, `SECURE`, `HTTP_ADDR`).
* Для локальной разработки → `APP_ENV=dev` (без HSTS, без Cache-Control).
* Для деплоя → `APP_ENV=prod` + HTTPS + реальный CSRF-ключ.

---

## ✅ **Проект сейчас готов:**

* к безопасному деплою в prod,
* с CSP, CSRF, HSTS, валидацией, flash и middleware,
* и с лёгкой архитектурой (`core`, `handler`, `templates`).



---
## итоговая структура

```
myApp/
│
├─ cmd/
│  └─ app/
│     └─ main.go                 # запуск HTTP-сервера, graceful shutdown, CSRF, HSTS
│
├──internal/
│  ├── core/
│  │    ├── server.go       // Фабрика http.Server с таймаутами
│  │    ├── config.go       // Конфигурация приложения (AppName, Addr, Env, Secure, ...)
│  │    ├── router.go       //  маршрутизация (chi.Router)
│  │    ├── common.go       // Базовые middleware (лог, recover, timeout, CSP)
│  │    ├── security.go     // Заголовки безопасности (CSP, XFO, MIME, Referrer)
│  │    ├── logger.go       // логирование (не реализовано)
│  │    ├── errors.go       // унифицированные ошибки (не реализовано)
│  │    └── response.go     // стандартные ответы (не реализовано)
│  │
│  └─ http/
│     └─ handler/
│        ├─ home.go              # главная страница
│        ├─ form.go              # форма + PRG-редирект
│        └─ about.go             # страница «О нас»
│
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
