
***Проект интернет магазина ( Go 1.25.1 )***

```
myApp/
├─ cmd/
│  └─ app/
│     └─ main.go                 # только сборка и wiring
│
├─ internal/
│  ├─ app/                       # высокоуровневая сборка/инициализация
│  │  └─ server.go               # запуск http-сервера, DI узлы
│  ├─ config/
│  │  └─ config.go               # загрузка конфигурации (env/flags)
│  ├─ adapter/                   # внешние адаптеры (transport, storage, etc.)
│  │  └─ http/
│  │     ├─ router.go
│  │     ├─ middleware.go
│  │     └─ handler/
│  │        ├─ health.go
│  │        └─ home.go
│  ├─ domain/                    # (опционально) бизнес-модели/интерфейсы
│  └─ usecase/                   # (опционально) бизнес-логика
│
├─ web/
│  ├─ assets/
│  │  └─ css/
│  │     └─ style.css
│  └─ templates/
│     ├─ layouts/
│     │  └─ base.gohtml          # предпочтителен .gohtml
│     ├─ partials/
│     │  ├─ nav.gohtml
│     │  └─ footer.gohtml
│     └─ pages/
│        └─ home.gohtml
│
├─ .env                          # только для локалки
├─ go.mod
└─ Makefile

```

## 🗂 Структура и назначение

### **cmd/app/main.go**

Точка входа приложения.
Загружает конфиг, создаёт роутер, поднимает HTTP-сервер с таймаутами и graceful shutdown.

---

### **internal/config/config.go**

Читает переменные окружения (`APP_NAME`, `APP_ENV`, `HTTP_ADDR`).
Формирует структуру Config с дефолтными значениями.

---

### **internal/platform/server.go**

Фабрика `http.Server` с безопасными таймаутами и ограничением заголовков.
Защита от медленных клиентов и базовых DoS-атак.

---

### **internal/transport/httpx/middleware.go**

Набор общих middleware:
gzip, timeout, recoverer, real IP, request ID, логгер, secure headers (CSP и др.).
Гарантирует безопасные ответы и наблюдаемость.

---

### **internal/transport/httpx/router.go**

Определяет маршруты:
`/`, `/healthz`, `/readyz`, `/metrics`, `/assets/*`.

| Endpoint   | Что проверяет                      | Кто использует               | Когда важен            |
| ---------- | ---------------------------------- | ---------------------------- | ---------------------- |
| `/healthz` | жив ли процесс                     | мониторинг, kubernetes       | всегда                 |
| `/readyz`  | готов ли к работе (БД, кэш и т.п.) | балансировщики, оркестраторы | при старте/перезапуске |
| `/metrics` | показатели производительности      | Prometheus, Grafana          | постоянно              |

Подключает middleware и обработчики ошибок 404/405.
Служит аналогом файла `routes/web.php` в Laravel.

---

### **internal/transport/httpx/handlers/**

Содержит "контроллеры" и хендлеры.

* **health.go** — эндпоинты `/healthz` и `/readyz` (пока без БД).
* **home.go** — SSR-страница «Главная»: подключает шаблоны, рендерит данные.

---

### **web/templates/**

HTML-шаблоны для SSR.

* **layouts/base.tmpl** — основной layout (общая структура страницы).
* **partials/nav.tmpl** — шапка и навигация.
* **partials/footer.tmpl** — футер (год через {{year}}).
* **pages/home.tmpl** — контент главной страницы.

---

### **web/assets/css/style.css**

Минимальные стили для layout и навигации.
Подключаются как статические файлы через `/assets/css/style.css`.

---

### **.env.example**

Шаблон переменных окружения:
имя приложения, окружение, порт.

---

### **Makefile**

Упрощённые команды:
`make run` — запуск,
`make tidy` — сбор зависимостей,
`make test` — прогон тестов.

---

### **go.mod**

Описание модуля и зависимостей (`chi`, `prometheus/client_golang`).

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

# Детализация зависимостей (минимальная):

* **2 зависит от 1** (инфраструктура нужна до БД).
* **3 зависит от 2** (рендер каталога из БД).
* **4 зависит от 3** (товары уже есть).
* **5 зависит от 4** (админка редактирует реальные сущности).
* **6 зависит от 3–5** (API поверх готовых сервисов/репо).
* **7 зависит от 1–6** (всё оборачиваем в прод-окружение).
* **8 зависит от 1–7** (полируем поверх готового).


# Риски и превентивные меры (кратко)

* **Сложность SQL/производительность:** берём `sqlc`, ранние индексы, EXPLAIN.
* **Безопасность форм:** `nosurf`, cookie `HttpOnly+Secure+SameSite`, строгая CSP.
* **Стабильность деплоя:** миграции отдельно от запуска, blue/green на этапе 7.
* **Технический долг:** линтер/тесты в CI с первого спринта.

---


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
