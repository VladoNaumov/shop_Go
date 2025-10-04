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
