# 🧱 myApp — минималистичное ядро веб-сервера на Go (Echo-style структура)

Чистая архитектура: `cmd → app → http → core → view`.
Функции из коробки: CSRF, CSP/HSTS, middleware-цепочка, статика с кэшем, форма с валидацией, healthcheck, 404, дневные логи с ротацией и graceful shutdown.

---

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
│        ├─ security.go            # CSP, XFO, Referrer, nosniff, Permissions, COOP, HSTS, CacheStatic, Keep-Alive
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

```

---

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

* `GET /form` — рендер + скрытое поле CSRF (`{{ .CSRFField }}`).
* `POST /form` — серверная валидация (`validator/v10`), на ошибках — повторный рендер с отображением `Errors`, на успехе — PRG: `303 /form?ok=1`.
* **Базы данных нет** — по твоему требованию.

---

## 🧾 Конфигурация (ENV)

| Переменная  | Описание                                            | Пример         |
| ----------- | --------------------------------------------------- | -------------- |
| `APP_NAME`  | Название приложения                                 | `myApp`        |
| `HTTP_ADDR` | Адрес сервера                                       | `:8080`        |
| `APP_ENV`   | Окружение                                           | `dev` / `prod` |
| `CSRF_KEY`  | Секрет для CSRF (любая строка, хешуется в 32 байта) | `change-me`    |
| `SECURE`    | Включить HTTPS/HSTS режим (true/false)              | `true`         |

> В `prod`: `CSRF_KEY` обязателен и не должен быть дефолтным.

---

## 🗂️ Логи

* Путь: `myApp/logs/`
* Формат файла: `DD-MM-YYYY.log` (+ опционально `errors-DD-MM-YYYY.log`)
* Инициализация: `core.InitDailyLog()` в `main.go`
* Авто-ротация в полночь: фоновая горутина в `main.go`
* Авто-очистка старых логов: внутри `InitDailyLog()` (например, старше 7 дней)

Проверка:

* добавить `log.Println("Приложение запущено")` и открыть `logs/<сегодня>.log`
* PowerShell: `Get-Content logs\DD-MM-YYYY.log -Wait`

---

## 🚀 Запуск

```bash
go run ./cmd/app
```

или:

```bash
APP_ENV=prod SECURE=true CSRF_KEY="secret" HTTP_ADDR=":8080" go run ./cmd/app
```

Windows (PowerShell):

```powershell
$env:APP_ENV="prod"; $env:SECURE="true"; $env:CSRF_KEY="secret"; go run ./cmd/app
```

---

## 🧪 Что проверить после запуска

* `GET /` — главная страница отрендерилась.
* `GET /about`
* `GET /form` — в HTML-форме есть `name="_csrf"` с токеном.
* `POST /form` — ошибки валидации подсвечиваются; на успехе — `303` → `/form?ok=1`.
* `GET /healthz` — `{"status":"ok"}`.
* `GET /assets/...` — статические файлы отдаются (в `prod` — с заголовками кэша).
* Логи пишутся в `logs/DD-MM-YYYY.log`.

---

## 🧭 Почему такая структура

* **Похожа на Echo**: единая точка сборки (`app.go`), рядом — middleware, маршруты, статика, 404.
* **Без фреймворка**: стандартный `net/http` + `chi`, никаких скрытых “магий”.
* **Ясное разделение слоёв**: `app` (сборка), `http` (транспорт), `core` (инфраструктура), `handler` (контроллеры), `web` (представление).

---

## ➕ Куда развивать дальше

* `/api/v1/*` + глобальный JSON-ошибкообработчик (используя `core.Fail()`).
* Авторизация: cookie-сессии или Bearer-токены.
* `core/config.go` → `Viper` (`config.yaml` + ENV).
* `internal/view` с кэшом шаблонов.
* Метрики Prometheus / OpenTelemetry.

---
