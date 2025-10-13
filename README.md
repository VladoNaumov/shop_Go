**Проект интернет магазина ( Go 1.25.1 )**

# 🧱 myApp — минималистичный веб-сервер на Go за NGINX

**myApp** — это учебный, но продакшен-готовый boilerplate веб-сервера на Go 1.25.1.
- Работает за NGINX (реверс-прокси), обеспечивая безопасность, производительность и масштабируемость.
- Поддерживает HTML-страницы, формы с валидацией, JSON-ответы и готов к расширению (API, аутентификация, БД).
- Архитектура — слоистая, близкая к Clean/Hexagonal, с фокусом на OWASP Top 10.

---

## 📁 Структура проекта

```
Конечно 👍 Вот в том же формате, как у тебя в примере:

myApp/
├─ cmd/
│  └─ app/
│     └─ main.go    # Точка входа приложения: загружает конфиг, инициализирует логи и CSRF, 
│                    создаёт и запускает HTTP-сервер, ожидает сигнал завершения и выполняет graceful shutdown
│ 
├─ internal/
│  ├─ app/
│  │  └─ app.go                     # Сборка: chi.Router, middleware, статика, маршруты, 404
│  │ 
│  ├─ core/
│  │  ├─ config.go                  # ENV-конфиг, проверки для prod
│  │  ├─ errors.go                  # AppError, фабрики (BadRequest, Internal)
│  │  ├─ response.go                # JSON(), Fail() (RFC7807)
│  │  └─ logfile.go                 # JSON-логи с ротацией (7 дней)
│  ├─ http/
│  │  ├─ handler/
│  │  │  ├─ home.go                 # / (HTML)
│  │  │  ├─ about.go                # /about (HTML)
│  │  │  ├─ form.go                 # /form (GET/POST, валидация, PRG)
│  │  │  └─ misc.go                 # /healthz (JSON), NotFound (404 HTML)
│  │  └─ middleware/
│  │     ├─ proxy.go                # TrustedProxy для NGINX (X-Forwarded-For, Proto)
│  │     └─ security.go             # CSP, XFO, nosniff, Referrer, Permissions, COOP, HSTS
│  └─ view/
│     └─ view.go                    # Централизованный рендер шаблонов
├─ web/
│  ├─ assets/                      # CSS/JS/изображения/шрифты
│  └─ templates/
│     ├─ layouts/base.gohtml       # Основной layout
│     ├─ partials/nav.gohtml       # Навигация
│     ├─ partials/footer.gohtml    # Футер
│     └─ pages/{home,about,form,404}.gohtml # Страницы
├─ logs/                           # DD-MM-YYYY.log, errors-DD-MM-YYYY.log
├─ nginx.conf                      # NGINX: TLS, rate limiting, кэш, сжатие
├─ go.mod
└─ go.sum


## 🌐 Маршруты

| Путь           | Описание                                    | Тип   |
|----------------|---------------------------------------------|-------|
| `/`            | Главная страница                            | HTML  |
| `/about`       | О проекте                                   | HTML  |
| `/form` (GET)  | Форма с CSRF и nonce                        | HTML  |
| `/form` (POST) | Валидация, санитизация, PRG (/form?ok=1)    | HTML  |
| `/healthz`     | {"status":"ok"} (доступ через NGINX)        | JSON  |
| `/assets/*`    | Статика (кэш и gzip в NGINX)                | Static|
| `/*`           | 404 Not Found (шаблон)                      | HTML  |



## 🔐 Безопасность

| Механизм                   | Где включён                     | Назначение                                      |
|----------------------------|---------------------------------|------------------------------------------------|
| **CSRF**                   | app.New() → gorilla/csrf        | Токен в шаблонах; Secure=true в prod           |
| **CSP**                    | middleware.SecureHeaders         | Self + cdn.jsdelivr.net; nonce для стилей      |
| **X-Frame-Options**        | middleware.SecureHeaders         | DENY (от clickjacking)                         |
| **X-Content-Type-Options** | middleware.SecureHeaders         | nosniff (от MIME-sniffing)                     |
| **Referrer-Policy**        | middleware.SecureHeaders         | no-referrer-when-downgrade                     |
| **Permissions-Policy**     | middleware.SecureHeaders         | Отключает camera, microphone, geolocation      |
| **COOP**                   | middleware.SecureHeaders         | same-origin (изоляция)                         |
| **HSTS**                   | middleware.HSTS (prod)          | HTTPS-only через NGINX                         |
| **Timeout**                | chi/middleware.Timeout(15s)     | Прерывает зависшие запросы                     |
| **Rate Limiting**          | NGINX (limit_req)               | 100 req/s, burst=200 (от DoS)                  |
| **Parameter Pollution**    | middleware.SecureHeaders         | Проверяет дубли query-параметров               |
| **Санитизация**            | handler/form.go → bluemonday    | Удаляет вредоносный HTML                       |
| **TLS**                    | NGINX (Let’s Encrypt)           | TLS 1.2+, безопасные шифры                     |
| **Trusted Proxy**          | middleware/proxy.go             | X-Forwarded-For, X-Real-IP, X-Forwarded-Proto  |



## 🧩 Работа формы `/form`
- **GET**: Рендер с CSRF-токеном и nonce (централизованно через `view.Render`).
- **POST**: Ограничение размера (1MB), санитизация (bluemonday), валидация (validator/v10).
- **Ошибки**: Ререндер с подсветкой (`{{.Data.Errors}}`).
- **Успех**: PRG-редирект (303, `/form?ok=1`).
- **Без БД/email**: Данные в памяти.



## ⚙️ Конфигурация (ENV)

| Переменная             | Описание                     | Дефолт / Пример |
|------------------------|------------------------------|-----------------|
| `APP_NAME`             | Название приложения          | `myApp`         |
| `HTTP_ADDR`            | Адрес сервера                | `:8080`         |
| `APP_ENV`              | Окружение                    | `dev` / `prod`  |
| `CSRF_KEY`             | Секрет для CSRF (≥32 байта)  | Генерируется    |
| `SECURE`               | Включить HTTPS/HSTS (NGINX)  | `false` / `true`|
| `SHUTDOWN_TIMEOUT`     | Таймаут shutdown             | `10s`           |
| `READ_HEADER_TIMEOUT`  | Таймаут чтения заголовков    | `5s`            |
| `READ_TIMEOUT`         | Таймаут чтения запроса       | `10s`           |
| `WRITE_TIMEOUT`        | Таймаут ответа               | `30s`           |
| `IDLE_TIMEOUT`         | Таймаут простоя              | `60s`           |



## ✅ Состояние проекта

| №  | Компонент                  | Статус | Комментарий                              |
|----|----------------------------|--------|------------------------------------------|
| 1  | CSRF                       | ✅     | Токены в формах                          |
| 2  | Безопасные заголовки       | ✅     | CSP с nonce, HSTS, XFO, nosniff          |
| 3  | TLS (NGINX)                | ✅     | HTTPS через NGINX (Let’s Encrypt)        |
| 4  | Graceful Shutdown          | ✅     | Таймаут из ENV                           |
| 5  | Валидация/Санитизация      | ✅     | validator/v10 + bluemonday               |
| 6  | Ошибки (RFC7807)           | ✅     | core.Fail с RequestID                    |
| 7  | JSON-логи                  | ✅     | Ротация, errors-DD-MM-YYYY.log           |
| 8  | 404 / Health               | ✅     | HTML 404, JSON healthcheck               |
| 9  | Rate Limiting (NGINX)      | ✅     | 100 req/s в NGINX                        |
| 10 | Trusted Proxy              | ✅     | X-Forwarded-For/Proto в middleware       |


## 🚀 Запуск
- **NGINX**: Настроить `nginx.conf` (TLS, rate limiting, кэш, gzip).
- **Go**: `go run ./cmd/app`.

| Команда       | Описание                          |
|---------------|-----------------------------------|
| `make run`    | Запустить проект                  |
| `make build`  | Собрать bin/app.exe               |
| `make start`  | Запустить бинарник                |
| `make clean`  | Удалить bin                       |
| `make test`   | Запустить тесты                   |
| `make lint`   | go fmt, go vet                    |

### Как запускать (cmd / PowerShell)
Перейди в папку проекта и запускай:

| Команда (cmd / PowerShell)          | Назначение                          | Примечание                                                |
| ----------------------------------- | ----------------------------------- | --------------------------------------------------------- |
| `.\make.bat run` <br>или `make run` | **Запуск приложения из исходников** | Запускает `go run ./cmd/app` на порту `:8080`             |
| `.\make.bat build`                  | **Сборка бинарника**                | Создаёт `bin\app.exe`                                     |
| `.\make.bat start`                  | **Запуск собранного бинарника**     | Запускает `bin\app.exe`, если он существует               |
| `.\make.bat clean`                  | **Очистка сборки**                  | Удаляет папку `bin`                                       |
| `.\make.bat test`                   | **Запуск тестов Go**                | Выполняет `go test ./... -v`                              |
| `.\make.bat lint`                   | **Проверка кода линтером**          | Запускает `golangci-lint`; если не найден — установит его |
| `.\make.bat tidy`                   | **Обновление зависимостей**         | Выполняет `go mod tidy`                                   |

---

TEST command: golangci-lint run

| Команда                                         | Что делает                                   |
| ----------------------------------------------- | -------------------------------------------- |
| `golangci-lint run`                             | Запускает **все активные линтеры**           |
| `golangci-lint run --fast`                      | Только быстрые и базовые проверки            |
| `golangci-lint run ./internal/core`             | Проверяет конкретную папку                   |
| `golangci-lint run --out-format=github-actions` | Форматирует вывод для CI/CD                  |
| `golangci-lint help linters`                    | Показывает все доступные линтеры (100+ штук) |


## 🛠️ Работа с NGINX
- **NGINX как реверс-прокси**:
  - TLS (Let’s Encrypt), rate limiting (100 req/s, burst=200), gzip, кэш статики (1 год).
  - Проксирование: `X-Forwarded-For`, `X-Real-IP`, `X-Forwarded-Proto` к Go (middleware/proxy.go).
  - Ограничение `/healthz` по IP (OWASP A04).
- **Почему**: NGINX снижает нагрузку на Go (TLS, кэш, DoS), усиливает безопасность (HSTS, CORS).

---
```
## 📦 Дополнительные расширение
- **API**: Добавить `/api/*` (JSON, core.JSON/Fail).
- **Аутентификация**: JWT/cookie (`internal/http/session`).
- **БД**: SQLite/PostgreSQL (`internal/store`) для форм, миграции
- **Метрики**: Prometheus (`/metrics`) + Grafana.
- **Админка**: `/admin/*` с CRUD, ограничение в NGINX.
- **CI/CD**: GitHub Actions (lint, test, build).
- **Тестирование**: httptest, govulncheck, OWASP ZAP.
- 
🧩 **Проект минималистичный, безопасный, продакшен-готовый за NGINX. Готов к API, БД, аутентификации.**

 **Тесты**

* Написать `httptest` для `FormIndex` и `FormSubmit`:

    * Проверка статуса 200/303/400
    * Проверка, что CSRF-токен есть в HTML
    * Проверка обработки ошибок формы
* Добавить unit-тест для `cacheStatic()` (заголовки кэша присутствуют)
* Использовать `testing.T.Run` для table-driven тестов

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