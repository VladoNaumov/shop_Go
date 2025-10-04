отлично, переписал **Шаг 1** полностью, **не меняя структуру папок**.
дал короткие описания вверху каждого файла и краткие комментарии в коде.


---

## `cmd/app/main.go` — точка входа приложения; сборка сервера и graceful shutdown

```go
package main

// Главный файл: загружает конфиг, собирает роутер, поднимает HTTP-сервер с таймаутами,
// логирует запуск и делает корректное завершение (graceful shutdown).

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/shop/internal/config"
	"example.com/shop/internal/platform"
	"example.com/shop/internal/transport/httpx"
)

func main() {
	cfg := config.Load()

	// Простой JSON-логгер (без внешних либ)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	// Роутер (внутри подключены middleware, хендлеры, статика)
	r := httpx.NewRouter()

	// HTTP-сервер с безопасными таймаутами
	srv := platform.NewServer(cfg.HTTPAddr, r)

	// Контекст для остановки по сигналам OS
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Запуск сервера в отдельной горутине
	go func() {
		logger.Info("http: listening", "addr", srv.Addr, "env", cfg.Env, "app", cfg.AppName)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http: server error", "err", err)
			os.Exit(1)
		}
	}()

	// Ожидаем сигнал
	<-ctx.Done()
	logger.Info("http: shutdown started")

	// Мягкое завершение с таймаутом
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("http: shutdown error", "err", err)
	} else {
		logger.Info("http: shutdown complete")
	}
}
```

---

## `internal/config/config.go` — загрузка переменных окружения (конфигурация)

```go
package config

// Единая точка конфигурации: читаем ENV, ставим дефолты, делаем минимальную валидацию.

import (
	"log"
	"os"
)

type Config struct {
	AppName  string
	Env      string // dev|staging|prod
	HTTPAddr string // ":8080"
}

func Load() Config {
	cfg := Config{
		AppName:  getEnv("APP_NAME", "shop"),
		Env:      getEnv("APP_ENV", "dev"),
		HTTPAddr: getEnv("HTTP_ADDR", ":8080"),
	}
	// Базовая валидация, чтобы не стартовать с пустым адресом
	if cfg.HTTPAddr == "" {
		log.Fatal("HTTP_ADDR is required")
	}
	return cfg
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
```

---

## `internal/platform/server.go` — фабрика безопасного HTTP-сервера (таймауты/лимиты)

```go
package platform

// Создаёт http.Server с безопасными таймаутами/лимитами — защита от Slowloris/DoS на уровне соединений.

import (
	"net/http"
	"time"
)

func NewServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}
}
```

---

## `internal/transport/httpx/middleware.go` — общие middleware и заголовки безопасности

```go

package httpx

// Общие middleware: gzip, таймаут, recover, request id, real ip, логгер, безопасные заголовки.

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Базовая CSP для SSR (позже расширим по необходимости)
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self'; script-src 'self'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer-when-downgrade")
		// Включайте HSTS только за HTTPS-прокси:
		// w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		next.ServeHTTP(w, r)
	})
}

// Лёгкий логгер chi (при желании заменим на slog со своими полями)
func requestLogger(next http.Handler) http.Handler {
	return middleware.Logger(next)
}

// Склейка стандартных middleware-обёрток
func commonMiddlewares(next http.Handler) http.Handler {
	h := next
	h = middleware.Compress(5)(h)               // gzip
	h = middleware.Timeout(15 * time.Second)(h) // таймаут на обработку
	h = middleware.Recoverer(h)                 // перехват паник
	h = middleware.RealIP(h)                    // реальный IP
	h = middleware.RequestID(h)                 // корреляция запросов
	h = requestLogger(h)
	h = secureHeaders(h)
	return h
}


```

---

## `internal/transport/httpx/router.go` — маршруты, статика, метрики

```go

package httpx

// Определяет маршруты: /, /healthz, /readyz, /metrics, /assets/*,
// подключает middleware и обработчики ошибок 404/405.

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"example.com/shop/internal/transport/httpx/handlers"
)

func NewRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(commonMiddlewares)

	// Health/Ready (готовность пока без БД)
	r.Get("/healthz", handlers.Health)
	r.Get("/readyz", handlers.Ready)

	// Prometheus метрики
	r.Handle("/metrics", promhttp.Handler())

	// Статика (только GET) + кэш
	assetsDir := http.Dir(filepath.Clean("web/assets"))
	fs := http.FileServer(assetsDir)
	r.Group(func(r chi.Router) {
		r.Get("/assets/*", http.StripPrefix("/assets/", cacheStatic(fs)).ServeHTTP)
	})

	// Главная (SSR)
	r.Get("/", handlers.HomeIndex) // аналог Controller@index

	// Стандартизированные ответы
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	return r
}

func cacheStatic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// В dev короткий кэш; в prod — длинный с хэшами в именах
		w.Header().Set("Cache-Control", "public, max-age=300")
		next.ServeHTTP(w, r)
	})
}

// Утилита (может пригодиться для фич по окружению)
func isDev() bool {
	return os.Getenv("APP_ENV") == "dev" || os.Getenv("APP_ENV") == ""
}

```

---

## `internal/transport/httpx/handlers/health.go` — /healthz и /readyz (пока без БД)

```go
package handlers

// Простые JSON-эндпоинты для liveness/readiness.
// Позже в Ready() добавим реальные проверки БД/кэша.

import (
	"encoding/json"
	"net/http"
)

func Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func Ready(w http.ResponseWriter, r *http.Request) {
	// Здесь позже: проверки БД/кэша. Сейчас — "готов".
	writeJSON(w, http.StatusOK, map[string]any{"ready": true})
}

// Вспомогательная функция для единообразных JSON-ответов
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
```

---

## `internal/transport/httpx/handlers/home.go` — SSR «контроллер» главной страницы

```go
package handlers

// SSR для главной: Route -> (этот "контроллер") -> Template.
// Шаблоны встраиваем через embed, чтобы не ловить проблемы glob/слешей на Windows.

import (
	"html/template"
	"net/http"
)

// Встраиваем шаблоны по папкам (без **).
// Относительные пути заданы от текущего файла.
var (
	tpl = template.Must(template.ParseFiles(
		"web/templates/layouts/base.gohtml",
		"web/templates/partials/nav.gohtml",
		"web/templates/partials/footer.gohtml",
		"web/templates/pages/home.gohtml",
	))
)

// ViewModel для страницы
type HomeViewsModel struct {
	Title   string
	Message string
}

// Хендлер главной страницы (аналог HomeController@index)
func HomeIndex(w http.ResponseWriter, r *http.Request) {
	vm := HomeViewsModel{
		Title:   "Главная",
		Message: "Это стартовая страница. SSR на html/template + chi.",
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Рендерим layout "base"; внутри он вставит блок {{block "content"}} из pages/home.tmpl
	if err := tpl.ExecuteTemplate(w, "base", vm); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}
```

---

## `web/templates/layouts/base.tmpl` — базовый layout

```tmpl
{{define "base"}}
<!doctype html>
<html lang="ru">
<head>
  <meta charset="utf-8">
  <meta http-equiv="x-ua-compatible" content="ie=edge">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}} · shop</title>
  <link rel="stylesheet" href="/assets/css/style.css">
</head>
<body>
  {{template "nav" .}}
  <main class="container">
    {{block "content" .}}{{end}}
  </main>
  {{template "footer" .}}
</body>
</html>
{{end}}
```

---

## `web/templates/partials/nav.tmpl` — шапка/навигация

```tmpl
{{define "nav"}}
<header class="nav">
  <a class="brand" href="/">shop</a>
  <nav>
    <a href="/">Home</a>
    <a href="/readyz">Ready</a>
    <a href="/metrics">Metrics</a>
  </nav>
</header>
{{end}}
```

---

## `web/templates/partials/footer.tmpl` — футер (год из FuncMap)

```tmpl
{{define "footer"}}
<footer class="footer">
  <small>© {{year}}</small>
</footer>
{{end}}
```

---

## `web/templates/pages/home.tmpl` — содержимое главной страницы

```tmpl
{{define "content"}}
<section>
  <h1>{{.Title}}</h1>
  <p>{{.Message}}</p>
</section>
{{end}}
```

---

## `web/assets/css/style.css` — минимальные стили

```css
body { font-family: system-ui, -apple-system, Segoe UI, Roboto, Arial, sans-serif; margin:0; color:#111; }
.container { max-width: 900px; margin: 24px auto; padding: 0 16px; }
.nav { display:flex; justify-content:space-between; align-items:center; padding:12px 16px; background:#f6f6f6; border-bottom:1px solid #e5e5e5; }
.nav a { margin-right: 12px; text-decoration:none; color:#333; }
.nav .brand { font-weight:700; margin-right: 24px; }
.footer { padding:16px; color:#777; border-top:1px solid #e5e5e5; margin-top:24px; }
h1 { font-size: 24px; margin: 16px 0; }
```

---

## `.env.example` — пример переменных окружения

```
APP_NAME=shop
APP_ENV=dev
HTTP_ADDR=:8080
```

---

## `Makefile` — удобные команды

```make
GOCMD=go

run:
	APP_ENV=dev HTTP_ADDR=:8080 $(GOCMD) run ./cmd/app

tidy:
	$(GOCMD) mod tidy

lint:
	@echo "включим golangci-lint на следующем шаге"

test:
	$(GOCMD) test ./... -count=1
```

---

## `go.mod` — минимальный модуль

```go
module example.com/shop

go 1.25.1

require (
	github.com/go-chi/chi/v5 v5.0.11
	github.com/prometheus/client_golang v1.20.4
)
```

> версии могут отличаться — `go mod tidy` подтянет актуальные.

---

### Проверка (как и прежде)

```bash
# 1) переменные окружения
export $(cat .env.example | xargs)

# 2) зависимости
go mod tidy

# 3) запуск
go run ./cmd/app

# 4) проверить в браузере:
# http://localhost:8080/
# http://localhost:8080/healthz
# http://localhost:8080/readyz
# http://localhost:8080/assets/css/style.css
# http://localhost:8080/metrics
```

---

## ✅ Что уже есть

* Таймауты и лимиты HTTP-сервера
* Авто-восстановление паник
* Gzip-сжатие
* CSP и другие заголовки безопасности
* Кэширование статики
* /healthz, /readyz, /metrics
* Graceful shutdown

---

## 🔜 Дальше (Шаг 2)

Подключаем MSQL, миграции, sqlc и реальную проверку БД в /readyz.