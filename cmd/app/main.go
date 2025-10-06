package main

// Он не занимается бизнес-логикой, а только настраивает, запускает и корректно завершает HTTP-сервер.
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

	httpx "example.com/shop/internal/adapter/http"
	"example.com/shop/internal/app"
	"example.com/shop/internal/config"
	/*
		context — для управления временем жизни процессов (например, остановки сервера).
		errors — для проверки типов ошибок.
		slog — современный структурированный логгер из стандартной библиотеки Go 1.21+.
		net/http — стандартный HTTP-сервер.
		os, os/signal, syscall — для обработки системных сигналов (Ctrl+C, SIGTERM).
		time — для таймаутов.
		config, app, http — твои внутренние пакеты.
	*/)

func main() {
	// Загрузка конфигурации
	cfg := config.Load()

	// Простой JSON-логгер
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	// Инициализация маршрутизатора
	// Роутер (внутри подключены middleware, хендлеры, статика)
	router := httpx.Router()

	// Создание HTTP-сервер с безопасными таймаутами
	srv := app.Server(cfg.HTTPAddr, router)

	// Обработка сигналов (остановка)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	//Запускаем сервер асинхронно, чтобы можно было параллельно ждать сигналов.
	//Если сервер завершился не по «нормальному» закрытию (http.ErrServerClosed),
	//то логируем ошибку и завершаем программу.
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

	// корректное завершение работы без обрыва пользователей.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("http: shutdown error", "err", err)
	} else {
		logger.Info("http: shutdown complete")
	}
}
