# ⚙️ Как это работает

| Компонент                                | Назначение                                                                                                                                                         |
| ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **cmd/app/main.go**                      | Загружает конфиг, настраивает JSON-логи с ротацией, создаёт handler, поднимает http.Server с TLS (если Secure=true), graceful shutdown с core.Close.              |
| **internal/app/app.go**                  | Собирает chi.Router: middleware (RequestID, RealIP, Logger, Recoverer, Timeout, SecureHeaders с nonce, rate limiting), CSRF, статика с кэшем, маршруты, 404.      |
| **internal/app/server.go**               | Фабрика http.Server с таймаутами из конфига, TLS-конфиг (MinVersion TLS1.2, безопасные шифры), graceful Shutdown.                                                 |
| **internal/core/config.go**              | Читает ENV (APP_NAME, HTTP_ADDR, APP_ENV, CSRF_KEY, SECURE, CertFile, KeyFile, таймауты), дефолты, проверки в prod (CSRFKey >=32 байт, TLS-файлы, Addr).         |
| **internal/core/errors.go**              | AppError с фабриками (BadRequest, NotFound, Internal…), ограничение Fields, From для конвертации ошибок.                                                           |
| **internal/core/response.go**            | Ответы: JSON, NoContent, Fail (RFC7807 с Type, Instance, RequestID).                                                                                              |
| **internal/core/logfile.go**             | JSON-логи в logs/DD-MM-YYYY.log (+ errors-DD-MM-YYYY.log для ERROR), ротация/очистка (7 дней), LogError с полями.                                               |
| **internal/http/handler/**               | Контроллеры: HTML-страницы с шаблонами, форма с валидацией (validator/v10 + bluemonday санитизация), health JSON, 404. Nonce/CSRF в шаблонах.                   |
| **internal/http/middleware/security.go** | Заголовки: CSP с nonce, XFO, nosniff, Referrer, Permissions, COOP, HSTS (в prod/HTTPS), кэш статики по типам, проверка Parameter Pollution.                      |

---

## 🌐 Маршруты

| Путь           | Описание                                                | Тип        |
| -------------- | ------------------------------------------------------- | ---------- |
| `/`            | Главная                                                 | HTML       |
| `/about`       | О проекте                                               | HTML       |
| `/form` (GET)  | Форма с CSRF и nonce                                    | HTML       |
| `/form` (POST) | Валидация/санитизация, PRG-редирект (/form?ok=1)        | HTML       |
| `/healthz`     | {"status":"ok"}                                         | JSON       |
| `/assets/*`    | Статика из web/assets (в prod — кэш по типам файлов)     | Static     |
| `/*`           | 404 Not Found (шаблон)                                  | HTML       |

---

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

---

## 🧩 Работа формы `/form`

* GET: Рендер шаблона с CSRF-токеном и nonce (из context).
* POST: Ограничение размера (1MB), TrimSpace + санитизация (bluemonday), валидация (validator/v10).
* Ошибки: Ререндер с сообщениями ({{ .Errors.name }}), логирование через LogError.
* Успех: PRG-редирект (303 /form?ok=1).
* **Без БД/email**: Всё в памяти.

---

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

---

## ✅ Состояние проекта

| №  | Компонент                  | Статус | Комментарий                              |
| -- | -------------------------- |--------|:-----------------------------------------|
| 1  | **CSRF**                   | ✅      | В формах, с токеном в шаблонах           |
| 2  | **Безопасные заголовки**   | ✅      | CSP с nonce, HSTS, etc.                  |
| 3  | **TLS / HTTPS**            | ✅      | В prod через CertFile/KeyFile            |
| 4  | **Graceful Shutdown**      | ✅      | С таймаутом из ENV, core.Close           |
| 5  | **Валидация / Санитизация**| ✅      | validator/v10 + bluemonday               |
| 6  | **Ошибки (RFC7807)**       | ✅      | Через core.Fail с RequestID              |
| 7  | **JSON-логи**              | ✅      | С полями, ротация                        |
| 8  | **404 / Health**           | ✅      | Реализованы                              |
| 9  | **Rate Limiting**          | ✅      | От DoS                                   |
| 10 | **Тесты**                  | 🚧     | Добавить httptest, govulncheck, lint     |

---

## 📦 Расширение

* Добавить: API (/api/* с JSON), аутентификацию (JWT/cookie), админку, метрики (Prometheus).
* Тестирование: OWASP ZAP для скана уязвимостей, govulncheck для зависимостей.
* CI/CD: GitHub Actions с lint/test/build/scan.
* Мониторинг: Grafana Loki для логов.

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

---

# По всему проекту (архитектурная сводка с привязкой к файлам)

**Точка входа и запуск**

* `cmd/app/main.go`: читает конфиг (`core/config.go`), инициализирует JSON-логи (`core/logfile.go`), собирает HTTP-приложение (`internal/app/app.go`), создаёт сервер (`internal/app/server.go`), запускает с TLS (если Secure=true) и гасит по сигналам с Shutdown.

**Конфигурация и инфраструктура**

* `internal/core/config.go`: ENV-конфиг + prod-валидация (CSRF_KEY, TLS-файлы, Addr).
* `internal/core/logfile.go`: JSON-логи + ротация/очистка.
* `internal/core/errors.go` + `internal/core/response.go`: AppError и RFC7807-ответы с логированием.

**Веб-приложение**

* `internal/app/app.go`: chi.Router, middleware (безопасность с nonce, rate limiting), CSRF, статика, маршруты и 404.
* `internal/app/server.go`: таймауты из ENV, TLS-конфиг.

**Безопасность и производительность**

* `internal/http/middleware/security.go`: CSP с nonce, заголовки, HSTS, кэш, Parameter Pollution check.

**Бизнес-маршруты и рендер**

* `internal/http/handler/`:

    * `home.go`, `about.go`: рендер HTML (layout + partials), CSRF.
    * `form.go`: форма (GET/POST), санитизация/валидация, PRG, без БД/email.
    * `misc.go`: `Health` JSON и `NotFound` 404-страница.

**Статика и шаблоны**

* Отдаются из `/assets` (`internal/app/app.go`), кэшируются в prod по типам.
* Шаблоны: `web/templates/layouts/`, `partials/`, `pages/` (с nonce и CSRF).

---

# Что уже хорошо с безопасностью (из коробки)

* **CSRF-защита** (`gorilla/csrf`): токен в формах, `SameSite=Lax`, `HttpOnly`, `Secure` в проде.
* **Жёсткие заголовки** (`middleware.SecureHeaders`): CSP с nonce, `X-Frame-Options=DENY`, `nosniff`, `Permissions-Policy`, `Referrer-Policy`, `COOP`.
* **HSTS** в проде (`middleware.HSTS`): принудительный HTTPS.
* **Безопасные таймауты** сервера: от slowloris.
* **Rate limiting** (100 req/s): от DoS.
* **Parameter Pollution check**: от обхода валидации.
* **Санитизация ввода** (`bluemonday`): от XSS/инъекций в формах.
* **Валидация** (`validator/v10`) + PRG: меньше повторных отправок.
* **Go templates**: авто-эскейпинг HTML (XSS-базис).
* **TLS**: Min 1.2, безопасные шифры в prod.
* **Логи без PII** по умолчанию.
* **Санити-чеки продакшена**: проверка ключей, TLS-файлов.

# На что обратить внимание / что добавить (по месту в проекте)

1. **Брутфорс-защита**
   Добавить middleware для форм/API (по IP).
   *Где*: `internal/http/middleware` + в `app/app.go`.

2. **Ограничение размера тела**
   Уже в `FormSubmit` (1MB); добавить в другие POST-хендлеры.
   *Где*: в будущих handlers.

3. **Строже CSP**
   Добавить hash для скриптов, мониторить violations.
   *Где*: `middleware.SecureHeaders`.

4. **Прокси и Secure режим**
   За NGINX/Apache: trust-proxy для X-Forwarded-Proto.
   *Где*: middleware в `app/app.go`.

5. **Сессии/куки (при логине)**
   `HttpOnly`, `Secure`, `SameSite=Strict`, TTL из ENV.
   *Где*: новый пакет `internal/http/session`.

6. **CORS (для API)**
   Ограничить origins/методы.
   *Где*: middleware в `internal/http/middleware`.

7. **Очистка ввода**
   Уже TrimSpace + bluemonday; добавить для Unicode/email.
   *Где*: в handlers.

8. **Скрытие инфры**
   Уже нет лишних заголовков; убрать Server в прокси.
   *Где*: сервер/прокси конфиг.

9. **Healthcheck доступ**
   Ограничить по IP/прокси.
   *Где*: middleware для `/healthz`.

10. **Приватность логов**
    Избегать PII в LogError.
    *Где*: в handlers/response.go.

11. **Ключи ротация**
    CSRF-ключ: криптослучайный, менять периодически.
    *Где*: ENV в prod.

Итог: архитектура "layered" близко к Clean/Hexagonal, с фокусом на безопасность. Базис (CSRF, XSS, DoS, TLS) покрыт. Для продакшена добавьте брутфорс-защиту, CORS, сессии — доведёт до enterprise-уровня. Тестируйте ZAP/Burp, мониторьте логи.
