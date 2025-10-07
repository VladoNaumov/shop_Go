package main

// Этот файл не содержит бизнес-логики.
// Он только настраивает, запускает и корректно завершает HTTP-сервер.

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpx "example.com/shop/internal/adapter/http"
	"example.com/shop/internal/app"
	"example.com/shop/internal/config"

	"github.com/gorilla/csrf"
)

func main() {
	// Загружаем конфигурацию приложения
	cfg := config.Load()

	// Настраиваем структурированный логгер
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Инициализация роутера (маршруты + middleware)
	router := httpx.NewRouter()

	// --- ✅ Подключаем CSRF middleware ---
	csrfKey := []byte("32-byte-long-auth-key") // длина ключа строго 32 байта
	csrfMw := csrf.Protect(
		csrfKey,
		csrf.Secure(cfg.Env == "production"), // true — только HTTPS в проде
		csrf.HttpOnly(true),
		csrf.Path("/"),
		csrf.SameSite(csrf.SameSiteLaxMode),
	)

	// Создаём HTTP-сервер, оборачивая роутер через CSRF-middleware
	srv := app.Server(cfg.HTTPAddr, csrfMw(router))

	// Контекст для graceful shutdown (Ctrl+C / SIGTERM)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// --- Асинхронный запуск сервера ---
	go func() {
		logger.Info("http: listening", "addr", srv.Addr, "env", cfg.Env, "app", cfg.AppName)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http: server error", "err", err)
			os.Exit(1)
		}
	}()

	// --- Ожидаем сигнал ---
	<-ctx.Done()
	logger.Info("http: shutdown started")

	// --- Корректное завершение работы ---
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("http: shutdown error", "err", err)
	} else {
		logger.Info("http: shutdown complete")
	}
}
