package main

// main.go

import (
	"context"
	"crypto/sha256"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"myApp/internal/core"

	"github.com/gorilla/csrf"
)

func main() {
	// --- 1. Загружаем конфиг и настраиваем логгер ---
	cfg := core.Load() // читаем настройки (порт, окружение, ключ CSRF и т.д.)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo, // уровень логирования: INFO и выше
	}))
	slog.SetDefault(logger)

	// sanity-checks для prod
	if cfg.Env == "prod" {
		if cfg.CSRFKey == "" {
			logger.Error("missing CSRF_KEY in prod")
			os.Exit(1) // фаталим прод без ключа
		}
		if !cfg.Secure {
			logger.Warn("APP_ENV=prod but Secure=false; HTTPS/HSTS disabled")
		}
	}

	// --- 2. Создаём маршрутизатор (роутер) ---
	router := core.NewRouter() // все обработчики и маршруты внутри пакета internal/http

	// --- 3. Оборачиваем роутер дополнительными мидлварами ---
	var h http.Handler = router

	// В продакшене включаем HSTS (строгая политика HTTPS)
	if cfg.Secure {
		h = hsts(h)
	}

	// --- 4. Настраиваем CSRF-защиту ---
	// Gorilla CSRF требует 32-байтовый ключ, берём SHA256 от строки из конфига.
	csrfKey := derive32(cfg.CSRFKey)
	h = csrf.Protect(
		csrfKey,
		csrf.Secure(cfg.Secure),             // только через HTTPS, если Secure = true
		csrf.SameSite(csrf.SameSiteLaxMode), // безопасный и удобный режим для форм
		csrf.HttpOnly(true),                 // токен не будет доступен из JS
		csrf.Path("/"),                      // CSRF-токен действует на весь сайт
	)(h)

	// --- 5. Создаём HTTP-сервер ---
	srv := core.Server(cfg.Addr, h)

	// --- 6. Настраиваем "graceful shutdown" (мягкое завершение) ---
	// Перехватываем сигналы ОС: Ctrl+C или SIGTERM (от Docker/сервиса)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// --- 7. Запускаем сервер в отдельной горутине ---
	go func() {
		logger.Info("http: listening", "addr", cfg.Addr, "env", cfg.Env, "app", cfg.AppName)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			// Ошибка запуска сервера — логируем и завершаем процесс
			logger.Error("http: server error", "err", err)
			os.Exit(1)
		}
	}()

	// --- 8. Ждём сигнал завершения ---
	<-ctx.Done()
	logger.Info("http: shutdown started")

	// --- 9. Завершаем сервер с таймаутом 10 секунд ---
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("http: shutdown error", "err", err)
	} else {
		logger.Info("http: shutdown complete")
	}
}

// derive32 создаёт 32-байтовый ключ из строки (SHA256).
// Используется для CSRF, так как библиотека требует ключ строго длиной 32 байта.
func derive32(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:] // возвращаем []byte длиной 32
}

// hsts добавляет HTTP-заголовок Strict-Transport-Security.
// Он заставляет браузеры всегда использовать HTTPS для этого домена.
// Активируется только в продакшене.
func hsts(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1 год действия, включает поддомены, preload (для добавления в браузерные списки)
		w.Header().Set("Strict-Transport-Security",
			"max-age=31536000; includeSubDomains; preload")
		next.ServeHTTP(w, r)
	})
}
