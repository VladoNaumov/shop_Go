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
)

func main() {
	// 1️⃣ Загружаем конфигурацию приложения и инициализируем логирование
	cfg := initConfig()

	// 2️⃣ Создаём контекст для фоновых задач (например, ротации логов)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 3️⃣ Запускаем ежедневную ротацию логов в отдельной горутине
	startLogRotation(ctx)

	// 4️⃣ Инициализируем HTTP-обработчик с CSRF-защитой (OWASP A02)
	handler := initHandler(cfg)

	// 5️⃣ Создаём HTTP-сервер с безопасными таймаутами (OWASP A05)
	srv := newHTTPServer(cfg, handler)

	// 6️⃣ Настраиваем перехват сигналов SIGINT/SIGTERM для graceful shutdown
	sigs, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 7️⃣ Запускаем HTTP-сервер в отдельной горутине
	runServer(srv, cfg)

	// 8️⃣ Ожидаем сигнал завершения и выполняем корректное выключение
	waitShutdown(sigs, srv, cfg)
}

// initConfig загружает конфигурацию приложения и подготавливает систему логов.
func initConfig() core.Config {
	cfg := core.Load()
	core.InitDailyLog()
	defer core.Close()
	return cfg
}

// startLogRotation запускает процесс ротации логов раз в сутки (24h).
func startLogRotation(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done(): // завершение по сигналу
				return
			case <-ticker.C:
				core.InitDailyLog()
			}
		}
	}()
}

// initHandler создаёт обработчик приложения с CSRF-защитой.
func initHandler(cfg core.Config) http.Handler {
	handler, err := app.New(cfg, derive32(cfg.CSRFKey))
	if err != nil {
		core.LogError("Ошибка инициализации приложения", map[string]interface{}{"error": err.Error()})
		os.Exit(1)
	}
	return handler
}

// newHTTPServer создаёт HTTP-сервер с таймаутами и обработчиком.
// (OWASP A05: Security Misconfiguration)
func newHTTPServer(cfg core.Config, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              cfg.Addr,              // Адрес сервера
		Handler:           h,                     // Обработчик запросов
		ReadHeaderTimeout: cfg.ReadHeaderTimeout, // Таймаут чтения заголовков
		ReadTimeout:       cfg.ReadTimeout,       // Таймаут чтения запроса
		WriteTimeout:      cfg.WriteTimeout,      // Таймаут записи ответа
		IdleTimeout:       cfg.IdleTimeout,       // Таймаут простоя
	}
}

// gracefulShutdown выполняет корректное завершение сервера за указанный таймаут.
func gracefulShutdown(srv *http.Server, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return srv.Shutdown(ctx)
}

// runServer запускает HTTP-сервер в отдельной горутине.
func runServer(srv *http.Server, cfg core.Config) {
	go func() {
		log.Printf("INFO: http: сервер запущен, addr=%s, env=%s, app=%s", cfg.Addr, cfg.Env, cfg.AppName)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			core.LogError("Ошибка работы сервера", map[string]interface{}{"error": err.Error()})
			os.Exit(1)
		}
	}()
}

// waitShutdown ожидает сигнал и выполняет graceful shutdown.
func waitShutdown(sigs context.Context, srv *http.Server, cfg core.Config) {
	<-sigs.Done()
	log.Println("INFO: http: начат процесс завершения")

	if err := gracefulShutdown(srv, cfg.ShutdownTimeout); err != nil {
		core.LogError("Ошибка завершения работы сервера", map[string]interface{}{"error": err.Error()})
	} else {
		log.Println("INFO: http: завершение работы выполнено")
	}
}

// derive32 генерирует 32-байтовый ключ для CSRF-защиты.
// Использует SHA-256 для детерминированной деривации из строки.
func derive32(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}
