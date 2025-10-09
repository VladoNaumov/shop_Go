
***Проект интернет магазина ( Go 1.25.1 )***


```markdown
# 🧱 myApp — минималистичное ядро веб-сервера на Go

**Цель:** чистое, безопасное и расширяемое ядро для веб-приложений на Go.  
Включает CSRF, HSTS, безопасные заголовки, шаблоны, формы и graceful shutdown.

---

## 📁 Структура проекта

```

myApp/
├─ cmd/
│  └─ app/
│     └─ main.go           # запуск HTTP-сервера, CSRF, HSTS, graceful shutdown
│
├─ internal/
│  └─ core/
│     ├─ config.go         # конфигурация приложения
│     ├─ server.go         # фабрика http.Server с таймаутами
│     ├─ router.go         # маршруты, статика, health-check
│     ├─ common.go         # middleware: лог, recover, timeout, CSP
│     ├─ security.go       # безопасные заголовки (CSP, XFO, Referrer, MIME)
│     ├─ errors.go         # унифицированные ошибки (AppError)
│     ├─ response.go       # стандартные ответы (JSON, error)
      └─ logfile.go        # 
│
├─ internal/http/handler/
│     ├─ home.go
│     ├─ about.go
│     ├─ form.go
│     └─ misc.go
│
└─ web/
├─ templates/
│  ├─ layouts/base.gohtml
│  ├─ partials/{nav,footer}.gohtml
│  └─ pages/{home,about,form,404}.gohtml
└─ assets/

````


## ⚙️ Как работает ядро

| Компонент       | Назначение                                                                            |
| --------------- | ------------------------------------------------------------------------------------- |
| **main.go**     | Загружает конфиг, включает CSRF, HTTPS/HSTS, запускает сервер с graceful shutdown.    |
| **config.go**   | Читает ENV (`APP_ENV`, `SECURE`, `CSRF_KEY`) и задаёт безопасные дефолты.             |
| **server.go**   | Создаёт `http.Server` с безопасными таймаутами (`Read=10s`, `Write=30s`, `Idle=60s`). |
| **router.go**   | Регистрирует маршруты и подключает статику; кэширует файлы в `prod`.                  |
| **common.go**   | Middleware: RequestID, RealIP, Logger, Recoverer, Timeout, SecureHeaders.             |
| **security.go** | CSP, XFO, MIME, Referrer, Permissions-Policy, COOP.                                   |
| **errors.go**   | Унифицированная структура `AppError`, фабрики `BadRequest`, `Internal` и др.          |
| **response.go** | Функции `JSON`, `NoContent`, `Fail` для API и ошибок.                                 |

---

## 🌐 Основные маршруты

| Путь           | Описание                                              | Тип    |
| -------------- | ----------------------------------------------------- | ------ |
| `/`            | Главная страница                                      | HTML   |
| `/about`       | О компании                                            | HTML   |
| `/form` (GET)  | Форма обратной связи с CSRF                           | HTML   |
| `/form` (POST) | Обработка формы, валидация, redirect `303 /form?ok=1` | HTML   |
| `/healthz`     | Проверка состояния сервера (JSON)                     | API    |
| `/assets/*`    | Статические файлы (`/web/assets`)                     | Static |
| `/*`           | Страница 404 Not Found                                | HTML   |

---

## 🔐 Безопасность

| Механизм                   | Где реализован                  | Назначение                           |
| -------------------------- | ------------------------------- | ------------------------------------ |
| **CSRF**                   | `main.go` (`csrf.Protect(...)`) | Автоматическая защита POST-форм.     |
| **CSP**                    | `security.go`                   | Разрешён только `self` и `jsdelivr`. |
| **X-Frame-Options**        | `security.go`                   | Запрет встраивания в iframe.         |
| **X-Content-Type-Options** | `security.go`                   | Защита от MIME-sniffing.             |
| **Referrer-Policy**        | `security.go`                   | Безопасная передача реферера.        |
| **HSTS**                   | `main.go → hsts()`              | Активируется, если `Secure=true`.    |
| **Timeout**                | `common.go`                     | Прерывает запросы дольше 15 секунд.  |

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
| -- | -------------------------- | ------ | ---------------------------------------- |
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

---

🧩 **Проект полностью рабочий, чистый и расширяемый.**
Подходит как основа для CMS, панели, или небольшого SaaS на Go.


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
│  │    ├── logfile.go       // логирование (не реализовано)
│  │    ├── errors.go       // унифицированные ошибки
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
