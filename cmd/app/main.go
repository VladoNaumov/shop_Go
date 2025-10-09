package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"myApp/internal/app"
	"myApp/internal/core"
)

func main() {
	// 1) Конфиг
	cfg := core.Load()

	// 2) Логи по дате + ежедневная ротация + автоочистка (7 дней)
	core.InitDailyLog()
	go func() {
		for {
			next := time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour)
			time.Sleep(time.Until(next))
			core.InitDailyLog()
		}
	}()

	// 3) Санити-проверки для prod
	if cfg.Env == "prod" {
		if cfg.CSRFKey == "" {
			log.Println("ERROR: missing CSRF_KEY in prod")
			os.Exit(1)
		}
		if !cfg.Secure {
			log.Println("WARN: APP_ENV=prod but Secure=false; HTTPS/HSTS disabled")
		}
	}

	// 4) Собираем http.Handler (router + middleware + CSRF + статика + 404)
	handler := app.New(cfg, derive32(cfg.CSRFKey))

	// 5) Создаём http.Server с безопасными таймаутами
	srv := app.Server(cfg.Addr, handler)

	// 6) Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("INFO: http: listening addr=%s env=%s app=%s", cfg.Addr, cfg.Env, cfg.AppName)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Printf("ERROR: http: server error: %v", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Println("INFO: http: shutdown started")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("ERROR: http: shutdown error: %v", err)
	} else {
		log.Println("INFO: http: shutdown complete")
	}
}

func derive32(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}
