***Проект интернет магазина ( Go 1.25.1 )***

# 🧱 myApp — минималистичное ядро веб-сервера на Go

---

Чистая архитектура: `cmd → app → http → core → view`.
Функции из коробки: CSRF, CSP/HSTS, middleware-цепочка, статика с кэшем, форма с валидацией, healthcheck, 404, дневные логи с ротацией и graceful shutdown.


## 📁 Структура проекта

``` 

myApp/
├─ cmd/
│  └─ app/
│     └─ main.go                   # entrypoint: конфиг, логи, graceful, запуск app.New()+Server
│
├─ internal/
│  ├─ app/
│  │  ├─ app.go                    # сборка приложения: chi.Router + middleware + статика + маршруты + 404
│  │  └─ server.go                 # http.Server с безопасными таймаутами
│  │
│  ├─ core/
│  │  ├─ config.go                 # конфиг (ENV), проверки для prod
│  │  ├─ errors.go                 # AppError, фабрики (BadRequest, NotFound, Internal…)
│  │  ├─ response.go               # JSON(), NoContent(), Fail() (RFC7807 style)
│  │  └─ logfile.go                # логи по датам, авто-ротация (опциональная) и очистка
│  │
│  └─ http/
│     ├─ handler/
│     │  ├─ common.go              # PageData, csrfField, render()
│     │  ├─ home.go                # /
│     │  ├─ about.go               # /about
│     │  ├─ form.go                # /form (GET/POST) + валидация validator/v10 + PRG
│     │  └─ misc.go                # /healthz (JSON), NotFound (404)
│     │
│     └─ middleware/
│        └─ security.go            # CSP, XFO, Referrer, nosniff, Permissions, COOP, HSTS, CacheStatic, Keep-Alive
│
├─ web/
│  ├─ assets/                      # статические файлы (CSS/JS/изображения/шрифты)
│  └─ templates/
│     ├─ layouts/base.gohtml
│     ├─ partials/{nav,footer}.gohtml
│     └─ pages/{home,about,form,404}.gohtml
│
├─ logs/                           # генерируется автоматически: DD-MM-YYYY.log (+ errors-DD-MM-YYYY.log, если включено)
├─ go.mod
└─ go.sum

````
## ⚙️ Как это работает

| Компонент                                | Назначение                                                                                                                                                         |
| ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **cmd/app/main.go**                      | Загружает конфиг, настраивает логи (по дате), включает ежедневную ротацию, создаёт `handler := app.New(cfg, csrfKey)`, поднимает `http.Server`, graceful shutdown. |
| **internal/app/app.go**                  | Собирает `chi.Router`: цепочка middleware, CSRF, статика, маршруты, 404. Это “сердце”, аналог `app.go` у Echo.                                                     |
| **internal/app/server.go**               | Фабрика `http.Server` с безопасными таймаутами: `ReadHeader=5s`, `Read=10s`, `Write=30s`, `Idle=60s`.                                                              |
| **internal/core/config.go**              | Читает ENV (`APP_NAME`, `HTTP_ADDR`, `APP_ENV`, `CSRF_KEY`, `SECURE`), даёт дефолты, sanity-проверки в prod.                                                       |
| **internal/core/errors.go**              | Единая модель ошибок `AppError` + фабрики (`BadRequest`, `Unauthorized`, `Forbidden`, `NotFound`, `Internal`).                                                     |
| **internal/core/response.go**            | Утилиты ответа: `JSON`, `NoContent`, `Fail` (RFC7807 ProblemDetail).                                                                                               |
| **internal/core/logfile.go**             | Создаёт `logs/DD-MM-YYYY.log`, при желании — `errors-DD-MM-YYYY.log`, очищает старше N дней, поддерживает ротацию в полночь.                                       |
| **internal/http/handler/**               | Контроллеры: HTML-страницы, форма, health, 404. Используют Go templates + CSRF.                                                                                    |
| **internal/http/middleware/security.go** | Безопасные заголовки (CSP, XFO, Referrer-Policy, nosniff, Permissions-Policy, COOP), HSTS (в prod/HTTPS), кэш для статики.                                         |

---

## 🌐 Маршруты

| Путь           | Описание                                            | Тип        |
| -------------- | --------------------------------------------------- | ---------- |
| `/`            | Главная                                             | HTML       |
| `/about`       | О проекте                                           | HTML       |
| `/form` (GET)  | Форма с CSRF                                        | HTML       |
| `/form` (POST) | Валидация (`validator/v10`), PRG (`303 /form?ok=1`) | HTML       |
| `/healthz`     | `{"status":"ok"}`                                   | API (JSON) |
| `/assets/*`    | Статика из `web/assets` (в prod — с кэшем)          | Static     |
| `/*`           | 404 Not Found (шаблон)                              | HTML       |

---

## 🔐 Безопасность

| Механизм                   | Где включён                            | Назначение                                                                    |
| -------------------------- | -------------------------------------- | ----------------------------------------------------------------------------- |
| **CSRF**                   | `app.New()` → `gorilla/csrf`           | Токен доступен в шаблоне как `{{ .CSRFField }}` через `csrf.TemplateField(r)` |
| **CSP**                    | `middleware.SecureHeaders`             | Явные источники: `self`, `jsdelivr` (для Bootstrap)                           |
| **X-Frame-Options**        | `SecureHeaders`                        | `DENY` — защита от clickjacking                                               |
| **X-Content-Type-Options** | `SecureHeaders`                        | `nosniff` — защита от MIME-sniffing                                           |
| **Referrer-Policy**        | `SecureHeaders`                        | `no-referrer-when-downgrade`                                                  |
| **Permissions-Policy**     | `SecureHeaders`                        | `camera=(), microphone=(), geolocation=(), payment=()`                        |
| **COOP**                   | `SecureHeaders`                        | `same-origin` — изоляция                                                      |
| **HSTS**                   | `middleware.HSTS` (если `Secure=true`) | `max-age=31536000; includeSubDomains; preload`                                |
| **Timeout**                | `chi/middleware.Timeout(15s)`          | Прерывает зависшие запросы                                                    |

---

## 🧩 Работа формы `/form`

* При `GET /form` сервер рендерит страницу с CSRF-полем.
* При `POST /form` выполняется валидация через `validator/v10`.
* При ошибках — повторный рендер с сообщениями (`{{ .Errors.email }}` и т.д.).
* При успехе — redirect `303 /form?ok=1` (паттерн PRG).
* **Запись в БД отсутствует** — всё выполняется в памяти (по твоему требованию).

---

## ⚙️ Конфигурация

| Переменная  | Описание            | Пример           |
| ----------- | ------------------- | ---------------- |
| `APP_NAME`  | Название приложения | `myApp`          |
| `HTTP_ADDR` | Адрес сервера       | `:8080`          |
| `APP_ENV`   | Окружение           | `dev` / `prod`   |
| `CSRF_KEY`  | Секрет для CSRF     | `supersecretkey` |
| `SECURE`    | Включить HTTPS/HSTS | `true`           |

---

```bash
go run ./cmd/app
````

```bash
make run
```

## ✅ Состояние проекта

| №  | Компонент                  | Статус | Комментарий                              |
| -- | -------------------------- |--------|:-----------------------------------------|
| 1  | **CSRF**                   | ✅      | Работает во всех формах                  |
| 2  | **Безопасные заголовки**   | ✅      | CSP, XFO, MIME, Referrer                 |
| 3  | **HSTS / HTTPS**           | ✅      | Через `cfg.Secure`                       |
| 4  | **Graceful Shutdown**      | ✅      | Через `context.WithTimeout` (10s)        |
| 5  | **Валидация форм**         | ✅      | Через `validator/v10`                    |
| 6  | **Унифицированные ошибки** | ✅      | `core/errors.go`                         |
| 7  | **JSON-ответы**            | ✅      | `core/response.go`                       |
| 8  | **Логирование**            | ✅      | Стандартный `chi` и `log.Printf`         |
| 9  | **404 / health-check**     | ✅      | Реализованы                              |
| 10 | **Тесты**                  | 🚧     | Планируется `httptest` и `golangci-lint` |

---

## 📦 Готов к расширению

Можно добавить:

* `/api/*` — JSON API (`core.JSON`, `core.Fail`)
* `/auth/*` — авторизация по cookie/token
* `/admin/*` — панель управления
* метрики Prometheus / OpenTelemetry

## как запустить проект ##
- go run ./cmd/app

| Команда        | Описание                                           |
| -------------- | -------------------------------------------------- |
| `.\make run`   | запустить проект                                   |
| `.\make build` | собрать бинарник `bin\app.exe`                     |
| `.\make start` | запустить бинарник                                 |
| `.\make clean` | удалить `bin`                                      |
| `.\make test`  | запустить все Go-тесты                             |
| `.\make lint`  | форматировать и проверить код (`go fmt`, `go vet`) |


---

🧩 **Проект полностью рабочий, чистый и расширяемый.**
Подходит как основа для CMS, панели, или небольшого SaaS на Go.

## 🚀 **Что осталось сделать (финальные задачи)**

- 1️⃣ добавить /api/v1/ с JSON-ответами через core.Fail() и core.JSON;
- 2️⃣ или аутентификацию с токенами (Bearer / Cookie);
- 3️⃣ или сделать авто-рендер шаблонов через централизованный view.Render() (ещё ближе к Laravel).

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
