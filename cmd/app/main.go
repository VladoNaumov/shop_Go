// cmd/app/main.go
package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"myApp/internal/app"
	"myApp/internal/core"
	"myApp/internal/storage"

	"golang.org/x/crypto/pbkdf2"
)

func main() {
	cfg := core.Load()
	core.LogInfo("Приложение запущено", map[string]interface{}{
		"env":    cfg.Env,
		"addr":   cfg.Addr,
		"secure": cfg.Secure,
		"app":    cfg.AppName,
	})

	core.InitDailyLog()

	db, err := storage.NewDB()
	if err != nil {
		core.LogError("Ошибка БД", map[string]interface{}{"error": err})
		os.Exit(1)
	}

	migrations := storage.NewMigrations(db)
	if err := migrations.RunMigrations(); err != nil {
		core.LogError("Ошибка миграций", map[string]interface{}{"error": err})
		os.Exit(1)
	}

	csrfKey := deriveSecureKey(cfg.CSRFKey)

	handler, err := app.New(cfg, db, csrfKey)
	if err != nil {
		core.LogError("Ошибка app.New", map[string]interface{}{"error": err})
		os.Exit(1)
	}

	srv := newHTTPServer(cfg, handler)

	sigs, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go runServer(srv, cfg)

	<-sigs.Done()
	core.LogInfo("Завершение...", nil)

	if err := srv.Shutdown(context.Background()); err != nil {
		core.LogError("Ошибка shutdown", map[string]interface{}{"error": err})
	}

	_ = storage.Close(db)
	core.Close()
}

func newHTTPServer(cfg core.Config, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           h,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}
}

func runServer(srv *http.Server, cfg core.Config) {
	core.LogInfo("Сервер запущен", map[string]interface{}{"addr": cfg.Addr})
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		core.LogError("Сервер упал", map[string]interface{}{"error": err})
		os.Exit(1)
	}
}

// deriveSecureKey — 32-байтовый ключ
func deriveSecureKey(secret string) []byte {
	if len(secret) == 0 {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			panic("unable to generate random bytes: " + err.Error())
		}
		return b
	}
	salt := []byte("myapp-session-salt")
	return pbkdf2.Key([]byte(secret), salt, 4096, 32, sha256.New)
}
