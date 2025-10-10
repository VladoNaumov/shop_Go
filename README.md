***Проект интернет магазина ( Go 1.25.1 )***

# 🧱 myApp — минималистичное ядро веб-сервера на Go

---

## 📁 Структура проекта

``` 

myApp/
├─ cmd/
│  └─ app/
│     └─ main.go                    # entrypoint: конфиг, логи, graceful, запуск app.New()+Server
│
├─ internal/
│  ├─ app/
│  │  ├─ app.go                     # сборка приложения: chi.Router + middleware + статика + маршруты + 404
│  │  └─ server.go                  # http.Server с безопасными таймаутами
│  │
│  ├─ core/
│  │  ├─ config.go                  # конфиг (ENV), проверки для prod
│  │  ├─ errors.go                  # AppError, фабрики (BadRequest, NotFound, Internal…)
│  │  ├─ response.go                # JSON(), NoContent(), Fail() (RFC 7807 style)
│  │  └─ logfile.go                 # логи по датам, авто-ротация и очистка
│  │
│  └─ http/
│     ├─ handler/
│     │  ├─ home.go                 # /
│     │  ├─ about.go                # /about
│     │  ├─ form.go                 # /form (GET/POST) + валидация validator/v10 + PRG
│     │  └─ misc.go                 # /healthz (JSON) и NotFound (404)
│     │
│     └─ middleware/
│        └─ security.go             # CSP, XFO, Referrer, nosniff, Permissions, COOP, HSTS, CacheStatic, Keep-Alive
│
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


# ⚙️ Как это работает

| Компонент                                | Назначение                                                                                                                                                         |
| ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **cmd/app/main.go**                      | Загружает конфиг, настраивает логи (по дате), включает ротацию, создаёт handler, поднимает http.Server с TLS (если Secure=true), graceful shutdown (с core.Close). |
| **internal/app/app.go**                  | Собирает chi.Router: middleware (вкл. rate limiting), CSRF, статика с кэшем, маршруты, 404. Генерирует nonce для CSP.                                           |
| **internal/app/server.go**               | Фабрика http.Server с таймаутами из конфига, TLS-конфигурацией (MinVersion TLS1.2, безопасные шифры).                                                             |
| **internal/core/config.go**              | Читает ENV (APP_NAME, HTTP_ADDR, APP_ENV, CSRF_KEY, SECURE, CertFile, KeyFile, таймауты), дефолты, проверки в prod (CSRFKey >=32 байт, TLS-файлы).               |
| **internal/core/errors.go**              | Модель AppError + фабрики (BadRequest, NotFound, etc.). Ограничивает Fields для валидации.                                                                        |
| **internal/core/response.go**            | Ответы: JSON, NoContent, Fail (RFC7807 с Type, Instance). Логирует с RequestID.                                                                                    |
| **internal/core/logfile.go**             | Логи в logs/DD-MM-YYYY.log (JSON-формат), errors-DD-MM-YYYY.log для ERROR, ротация/очистка (7 дней), LogError с полями.                                        |
| **internal/http/handler/**               | Контроллеры: HTML-страницы с шаблонами, форма с валидацией (validator/v10 + bluemonday), health (JSON), 404. Nonce/CSRF в шаблонах.                              |
| **internal/http/middleware/security.go** | Заголовки: CSP с nonce, XFO, nosniff, Referrer, Permissions, COOP, HSTS (в prod/HTTPS), кэш статики, проверка Parameter Pollution.                               |



## 🌐 Маршруты

| Путь           | Описание                                                | Тип        |
| -------------- | ------------------------------------------------------- | ---------- |
| `/`            | Главная                                                 | HTML       |
| `/about`       | О проекте                                               | HTML       |
| `/form` (GET)  | Форма с CSRF и nonce                                    | HTML       |
| `/form` (POST) | Валидация, санитизация, PRG-редирект (/form?ok=1)       | HTML       |
| `/healthz`     | {"status":"ok"}                                         | JSON       |
| `/assets/*`    | Статика из web/assets (в prod — кэш по типам файлов)     | Static     |
| `/*`           | 404 Not Found (шаблон)                                  | HTML       |



## 🔐 Безопасность

| Механизм                   | Где включён                            | Назначение                                                                    |
| -------------------------- | -------------------------------------- | ----------------------------------------------------------------------------- |
| **CSRF**                   | app.New() → gorilla/csrf               | Токен в шаблонах как {{ .CSRFField }}; Secure=true в prod                     |
| **CSP**                    | middleware.SecureHeaders               | Self + cdn.jsdelivr.net; nonce для inline-стилей                              |
| **X-Frame-Options**        | SecureHeaders                          | DENY — от clickjacking                                                        |
| **X-Content-Type-Options** | SecureHeaders                          | nosniff — от MIME-sniffing                                                    |
| **Referrer-Policy**        | SecureHeaders                          | no-referrer-when-downgrade                                                    |
| **Permissions-Policy**     | SecureHeaders                          | Отключает camera, microphone, geolocation, payment                            |
| **COOP**                   | SecureHeaders                          | same-origin — изоляция                                                        |
| **HSTS**                   | middleware.HSTS (если Secure=true)     | max-age=31536000; includeSubDomains; preload                                  |
| **Timeout**                | chi/middleware.Timeout(15s)            | Прерывает зависшие запросы                                                    |
| **Rate Limiting**          | app.New() → rateLimit                  | 100 req/s — от DoS                                                            |
| **Parameter Pollution**    | SecureHeaders                          | Проверяет дубли query-параметров                                              |
| **Санитизация**            | handler/form.go → bluemonday           | Удаляет вредоносный HTML в формах                                             |
| **TLS**                    | main.go / server.go (если Secure=true) | Min TLS1.2, безопасные шифры; CertFile/KeyFile из ENV                         |



## 🧩 Работа формы `/form`

* GET: Рендер шаблона с CSRF-токеном и nonce.
* POST: Ограничение размера (1MB), санитизация (bluemonday), валидация (validator/v10).
* Ошибки: Ререндер с сообщениями ({{ .Errors.name }}).
* Успех: PRG-редирект (303 /form?ok=1).
* **Без БД/email**: Всё в памяти.



## ⚙️ Конфигурация (ENV)

| Переменная             | Описание                     | Дефолт / Пример          |
| ---------------------- | ---------------------------- | ------------------------ |
| `APP_NAME`             | Название приложения          | `myApp`                  |
| `HTTP_ADDR`            | Адрес сервера                | `:8080`                  |
| `APP_ENV`              | Окружение                    | `dev` / `prod`           |
| `CSRF_KEY`             | Секрет для CSRF (min 32 байт)| Генерируется в dev       |
| `SECURE`               | Включить HTTPS/HSTS/TLS      | `false` / `true`         |
| `TLS_CERT_FILE`        | Путь к сертификату           | `` (для prod)            |
| `TLS_KEY_FILE`         | Путь к ключу                 | `` (для prod)            |
| `SHUTDOWN_TIMEOUT`     | Таймаут shutdown             | `10s`                    |
| `READ_HEADER_TIMEOUT`  | Таймаут чтения заголовков    | `5s`                     |
| `READ_TIMEOUT`         | Таймаут чтения запроса       | `10s`                    |
| `WRITE_TIMEOUT`        | Таймаут ответа               | `30s`                    |
| `IDLE_TIMEOUT`         | Таймаут простоя              | `60s`                    |



## ✅ Состояние проекта

| №  | Компонент                  | Статус | Комментарий                              |
| -- | -------------------------- |--------|:-----------------------------------------|
| 1  | **CSRF**                   | ✅      | В формах, с токеном в шаблонах           |
| 2  | **Безопасные заголовки**   | ✅      | CSP с nonce, HSTS, etc.                  |
| 3  | **TLS / HTTPS**            | ✅      | В prod через CertFile/KeyFile            |
| 4  | **Graceful Shutdown**      | ✅      | С таймаутом из ENV, core.Close           |
| 5  | **Валидация / Санитизация**| ✅      | validator/v10 + bluemonday               |
| 6  | **Ошибки (RFC7807)**       | ✅      | Через core.Fail с RequestID              |
| 7  | **JSON-логи**              | ✅      | JSON с полями, ротация                   |
| 8  | **404 / Health**           | ✅      | Реализованы                              |
| 9  | **Rate Limiting**          | ✅      | От DoS                                   |
| 10 | **Тесты**                  | 🚧     | Добавить httptest, golangci-lint         |



## 📦 Расширение

* Добавить: API (/api/* с JSON), аутентификацию (JWT/cookie), админку, метрики (Prometheus).
* Тестирование: govulncheck для зависимостей, OWASP ZAP для скана.
* CI/CD: GitHub Actions с lint/test/build.

## 🚀 Запуск

- `go run ./cmd/app`

| Команда        | Описание                                           |
| -------------- | -------------------------------------------------- |
| `make run`     | Запустить проект                                   |
| `make build`   | Собрать бинарник bin/app.exe                       |
| `make start`   | Запустить бинарник                                 |
| `make clean`   | Удалить bin                                        |
| `make test`    | Запустить тесты                                    |
| `make lint`    | Форматировать/проверить код (go fmt, go vet)       |

🧩 **Проект безопасный, чистый, готов к масштабу.**

``` 



### 

### 🚀 **Что осталось сделать (финальные задачи)**

- 1 аутентификацию с токенами (Bearer / Cookie);
- 2 сделать авто-рендер шаблонов через централизованный view.Render() (ещё ближе к Laravel).

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

компактный **план-график** по спринтам с зависимостями, пакетами и критериями “готово”.

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
