package main

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

	"myApp/internal/app"
	"myApp/internal/config"
	httpx "myApp/internal/http"

	"github.com/gorilla/csrf"
)

func main() {
	// config + logger
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	// router
	router := httpx.NewRouter()

	// HSTS только в prod (требует HTTPS)
	var h http.Handler = router
	if cfg.Secure {
		h = hsts(h)
	}

	// CSRF (ключ делаем ровно 32 байта)
	csrfKey := derive32(cfg.CSRFKey)
	h = csrf.Protect(
		csrfKey,
		csrf.Secure(cfg.Secure),             // prod => только HTTPS
		csrf.SameSite(csrf.SameSiteLaxMode), // адекватно для форм
		csrf.HttpOnly(true),
		csrf.Path("/"),
	)(h)

	// server
	srv := app.Server(cfg.Addr, h)

	// graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("http: listening", "addr", cfg.Addr, "env", cfg.Env, "app", cfg.AppName)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http: server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("http: shutdown started")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("http: shutdown error", "err", err)
	} else {
		logger.Info("http: shutdown complete")
	}
}

func derive32(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:] // 32 байта
}

// простой HSTS-мидлвар только для prod (HTTPS обязателен)
func hsts(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1 год, включая поддомены; закомментируй preload, если не используешь
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		next.ServeHTTP(w, r)
	})
}
