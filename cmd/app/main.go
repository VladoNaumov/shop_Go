package main

//main.go
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
	"myApp/internal/storage"

	"github.com/jmoiron/sqlx"
)

func main() {
	// 1) Конфиг и логи
	config := core.Load()
	log.Printf("INFO: Secure=%v, Env=%s", config.Secure, config.Env)
	core.InitDailyLog()

	// 2) БД (с ретраями внутри storage.NewDB)
	db, err := storage.NewDB()
	if err != nil {
		core.LogError("Ошибка инициализации MySQL", map[string]interface{}{"error": err.Error()})
		os.Exit(1)
	}

	// 3) Миграции
	migrations := storage.NewMigrations(db)
	if err := migrations.RunMigrations(); err != nil {
		core.LogError("Ошибка выполнения миграций", map[string]interface{}{"error": err.Error()})
		os.Exit(1)
	}

	// 4) Контекст для фоновых задач (ротация логов)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 5) Ежедневная ротация логов
	startLogRotation(ctx)

	// 6) Инициализируем Gin-обработчик с CSRF и DB
	handler := initHandler(config, db)

	// 7) HTTP-сервер с таймаутами (OWASP A05)
	srv := newHTTPServer(config, handler)

	// 8) Перехват сигналов
	sigs, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 9) Запуск сервера
	runServer(srv, config)

	// 10) Ожидаем сигнал завершения
	waitShutdown(sigs, srv, config)

	// 11) Закрытие ресурсов
	if cerr := storage.Close(db); cerr != nil {
		core.LogError("Ошибка закрытия MySQL", map[string]interface{}{"error": cerr.Error()})
	}
	core.Close()
}

// startLogRotation — ротация раз в сутки
func startLogRotation(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				core.InitDailyLog()
			}
		}
	}()
}

// initHandler — сборка приложения (получаем http.Handler = *gin.Engine)
func initHandler(cfg core.Config, db *sqlx.DB) http.Handler {
	handler, err := app.New(cfg, db, derive32(cfg.CSRFKey))
	if err != nil {
		core.LogError("Ошибка инициализации приложения", map[string]interface{}{"error": err.Error()})
		os.Exit(1)
	}
	return handler
}

// newHTTPServer — задаём таймауты сервера
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

// gracefulShutdown — корректное завершение
func gracefulShutdown(srv *http.Server, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return srv.Shutdown(ctx)
}

// runServer — запуск (ListenAndServe) в горутине
func runServer(srv *http.Server, cfg core.Config) {
	go func() {
		log.Printf("INFO: http: сервер запущен, addr=%s, env=%s, app=%s", cfg.Addr, cfg.Env, cfg.AppName)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			core.LogError("Ошибка работы сервера", map[string]interface{}{"error": err.Error()})
			os.Exit(1)
		}
	}()
}

// waitShutdown — ожидание сигналов и shutdown
func waitShutdown(sigs context.Context, srv *http.Server, cfg core.Config) {
	<-sigs.Done()
	log.Println("INFO: http: начат процесс завершения")
	if err := gracefulShutdown(srv, cfg.ShutdownTimeout); err != nil {
		core.LogError("Ошибка завершения сервера", map[string]interface{}{"error": err.Error()})
	} else {
		log.Println("INFO: http: завершение выполнено")
	}
}

// derive32 — 32-байтовый ключ CSRF из секрета (OWASP A02)
func derive32(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}
