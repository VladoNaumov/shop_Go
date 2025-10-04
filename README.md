

```
shop/
├─ cmd/
│  └─ app/
│     └─ main.go
├─ internal/
│  ├─ config/
│  │  └─ config.go
│  ├─ platform/
│  │  └─ server.go
│  └─ transport/
│     └─ httpx/
│        ├─ router.go
│        ├─ middleware.go
│        └─ handlers/
│           ├─ health.go
│           └─ home.go
├─ web/
│  ├─ assets/
│  │  └─ css/
│  │     └─ style.css
│  └─ templates/
│     ├─ layouts/
│     │  └─ base.tmpl
│     ├─ partials/
│     │  ├─ nav.tmpl
│     │  └─ footer.tmpl
│     └─ pages/
│        └─ home.tmpl
├─ .env.example
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
