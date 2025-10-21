# 🧱 **myApp — минималистичный веб-сервер на Go 1.25.1 за NGINX**

**myApp** — учебный, но продакшен-готовый boilerplate-проект на **Go 1.25.1**.
Подходит для разработки интернет-магазина, корпоративных сайтов и API-сервисов.

---

## ⚡ Основные возможности

* 🌀 Основан на **Gin** — быстром, безопасном фреймворке с middleware-цепочками.
* 🔒 Поддерживает **CSRF**, **CSP с nonce**, **HSTS**, **COOP**, **Referrer-Policy**.
* 📦 Работает за **NGINX** (реверс-прокси): TLS, rate-limit, gzip, кэш.
* 🧩 Слоистая архитектура (Core / App / Storage / HTTP / View) ≈ Clean Architecture.
* 🧱 Поддержка MySQL (через sqlx), шаблонов Go, и централизованных логов.
* 🧠 Соответствует рекомендациям OWASP Top 10.

---

## 📁 **Структура проекта**

```
myApp/
├─ cmd/
│  └─ app/
│     └─ main.go              # Точка входа (ENV, CSRF-key, DB, graceful shutdown)
│
├─ internal/
│  ├─ app/
│  │  └─ app.go               # Gin router, middleware, статика, маршруты
│  │
│  ├─ core/
│  │  ├─ config.go            # ENV-конфиг, Secure-режим, таймауты
│  │  ├─ context.go           # CtxNonce, контекстные ключи
│  │  ├─ errors.go            # AppError (RFC 7807)
│  │  ├─ response.go          # JSON(), Fail() — единый JSON-ответ
│  │  ├─ logger.go            # zerolog-логи + ротация файлов
│  │  └─ security.go          # CSP, HSTS, заголовки безопасности
│  │
│  ├─ storage/                # Работа с MySQL
│  │  ├─ db.go                # sqlx.DB, контекст, Close()
│  │  ├─ migrations.go        # Автомиграции
│  │  └─ products_repo.go     # Product, ListAll, GetByID
│  │
│  ├─ http/
│  │  └─ handler/
│  │     ├─ home.go           # /
│  │     ├─ about.go          # /about
│  │     ├─ form.go           # /form GET / POST
│  │     ├─ catalog.go        # /catalog
│  │     ├─ product.go        # /product/:id
│  │     ├─ notfound.go       # 404
│  │     └─ debug.go          # /debug (JSON)
│  │
│  └─ view/
│     └─ templates.go         # Централизованный рендер HTML-шаблонов
│
├─ migrations/
│  └─ 001_schema.sql          # Создание таблиц и демо-товаров
│
├─ web/
│  ├─ assets/                 # CSS/JS/шрифты/изображения
│  └─ templates/
│     ├─ layouts/base.gohtml
│     ├─ partials/{nav,footer}.gohtml
│     └─ pages/{home,about,form,catalog,product,404}.gohtml
│
├─ logs/                      # info- и error-логи с датой
├─ nginx.conf                 # Готовый reverse-proxy (TLS, gzip, cache)
├─ go.mod / go.sum
└─ Makefile / make.bat



## 🌐 **Маршруты**

| Путь           | Описание                    | Тип    |
| -------------- | --------------------------- | ------ |
| `/`            | Главная страница            | HTML   |
| `/about`       | О проекте                   | HTML   |
| `/form` GET    | Форма с CSRF и nonce        | HTML   |
| `/form` POST   | Валидация, санитизация, PRG | HTML   |
| `/catalog`     | Каталог из MySQL            | HTML   |
| `/product/:id` | Страница товара             | HTML   |
| `/debug`       | JSON ответ (health/info)    | JSON   |
| `/assets/*`    | Статика (CSS, JS, img)      | Static |
| `/*`           | 404 Not Found               | HTML   |

---

## 🔐 **Безопасность**

| Механизм               | Реализация / Middleware      | Назначение                        |
| ---------------------- | ---------------------------- | --------------------------------- |
| **CSRF**               | utrack/gin-csrf              | Токен в формах, Secure в prod     |
| **CSP + nonce**        | core.SecureHeaders           | Защита inline-скриптов            |
| **X-Frame-Options**    | core.SecureHeaders           | `DENY` (от clickjacking)          |
| **X-Content-Type**     | core.SecureHeaders           | `nosniff`                         |
| **Referrer-Policy**    | core.SecureHeaders           | `strict-origin-when-cross-origin` |
| **Permissions-Policy** | core.SecureHeaders           | Отключены camera, microphone, geo |
| **COOP**               | core.SecureHeaders           | `same-origin` изоляция            |
| **HSTS**               | core.HSTS (в prod)           | Принудительный HTTPS              |
| **Timeout**            | app.RequestTimeout           | Прерывает зависшие запросы        |
| **Rate Limit**         | NGINX (limit_req)            | Защита от DoS                     |
| **Sanitization**       | handler/form.go → bluemonday | Очистка HTML                      |
| **TLS**                | NGINX + Let’s Encrypt        | HTTPS, шифры TLS 1.2+             |
| **Trusted Proxies**    | r.SetTrustedProxies()        | Проверка X-Forwarded-For/Proto    |

---

## 🧩 **Форма `/form`**

* **GET** — отображение формы с CSRF и nonce
* **POST** — валидация (validator/v10), санитизация (bluemonday)
* **Ошибки** — ререндер формы с сообщениями
* **Успех** — PRG-редирект (303 → /form?ok=1)
* **Без БД/email** — чистый пример валидации и рендера

---

## ⚙️ **ENV-конфигурация**

| Переменная         | Назначение               | Пример       |
| ------------------ | ------------------------ | ------------ |
| `APP_NAME`         | Название приложения      | `myApp`      |
| `HTTP_ADDR`        | Адрес прослушивания      | `:8080`      |
| `APP_ENV`          | Окружение (dev/prod)     | `dev`        |
| `SECURE`           | Включить HSTS (для prod) | `true`       |
| `CSRF_KEY`         | Секрет (32 байта +)      | генерируется |
| `READ_TIMEOUT`     | Таймаут чтения запроса   | `10s`        |
| `WRITE_TIMEOUT`    | Таймаут ответа           | `30s`        |
| `IDLE_TIMEOUT`     | Таймаут простоя          | `60s`        |
| `SHUTDOWN_TIMEOUT` | Время graceful shutdown  | `10s`        |

---

## ✅ **Состояние проекта**

| Компонент               | Статус | Комментарий                       |
| ----------------------- | ------ | --------------------------------- |
| CSRF (utrack)           | ✅      | Работает через middleware         |
| CSP с nonce             | ✅      | SecureHeaders + nonce в шаблонах  |
| HSTS (prod)             | ✅      | Включается по флагу `SECURE=true` |
| Graceful Shutdown       | ✅      | Контекст + таймаут                |
| Валидация / Санитизация | ✅      | validator/v10 + bluemonday        |
| JSON-логи               | ✅      | zerolog + ротация файлов          |
| 404 / Debug             | ✅      | HTML + JSON healthcheck           |
| NGINX rate limit        | ✅      | 100 req/s + burst 200             |
| Trusted Proxy           | ✅      | SetTrustedProxies + X-Forwarded   |

---

## 🚀 **Запуск**

| Команда            | Описание                       |
| ------------------ | ------------------------------ |
| `.\make.bat run`   | Запуск из исходников           |
| `.\make.bat build` | Сборка бинарника (bin/app.exe) |
| `.\make.bat start` | Запуск собранного бинарника    |
| `.\make.bat clean` | Очистка папки bin              |
| `.\make.bat test`  | Запуск тестов Go               |
| `.\make.bat lint`  | Линтинг golangci-lint          |
| `.\make.bat tidy`  | Обновление зависимостей        |

---

## 🧮 **План-график (спринты)**

| Спринт | Цель                             | Пакеты / Компоненты                 | Критерии «Готово»                             |
| ------ | -------------------------------- | ----------------------------------- | --------------------------------------------- |
| **1**  | 🧱 Базовая архитектура и шаблоны | core, view, handler(home/about)     | Сервер запускается, шаблоны рендерятся OK     |
| **2**  | 🔐 Безопасность и middleware     | core/security, app.RequestTimeout   | CSP, CSRF, XFO, Timeout работают без ошибок   |
| **3**  | 🗄️ Подключение MySQL (каталог)  | storage/db, storage/products_repo   | /catalog отдаёт данные из БД                  |
| **4**  | 📮 Формы и валидация             | handler/form.go, view.Render        | /form POST валидирует и PRG-редирект OK       |
| **5**  | 📊 Логирование и ошибки          | core/logger, core/errors, core/Fail | Логи в файлах, ошибки в JSON формате          |
| **6**  | 🧰 Тестирование и линтинг        | all                                 | httptest, lint, govulncheck — без ошибок      |
| **7**  | 🧱 NGINX интеграция и TLS        | nginx.conf                          | HTTPS через Let's Encrypt, gzip, cache, limit |
| **8**  | 🔗 Расширения (API, Auth, Admin) | new packages                        | /api/* возвращает JSON, Auth через JWT/cookie |

```

---

## 🧠 **Дальнейшие расширения**

* `/api/*` — JSON API через core.JSON/Fail
* JWT / Session-Auth (через gin-sessions)
* `/metrics` — Prometheus метрики
* `/admin/*` — CRUD панель (с авторизацией)
* CI/CD — GitHub Actions (lint + test + build)
* Тестирование — `httptest`, `govulncheck`, OWASP ZAP

---

🧩 **Итог:**
Минималистичный, безопасный и расширяемый Go-проект,
готовый к продакшену за NGINX — чистая архитектура, CSRF + CSP + HSTS,
JSON-логи и шаблоны с nonce.



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