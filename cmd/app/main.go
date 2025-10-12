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
	// Загружает конфигурацию приложения из переменных окружения
	cfg := core.Load()

	// Инициализирует ротацию лог-файлов
	core.InitDailyLog()

	// Закрывает файлы логов при завершении приложения
	defer core.Close()

	// Создаёт контекст для управления ротацией логов
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запускает ежедневную ротацию логов в отдельной горутине
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

	// Инициализирует HTTP-обработчик с CSRF-защитой
	handler, err := app.New(cfg, derive32(cfg.CSRFKey))
	if err != nil {
		core.LogError("Ошибка инициализации приложения", map[string]interface{}{"error": err.Error()})
		os.Exit(1)
	}

	// Создаёт HTTP-сервер с заданной конфигурацией
	srv, err := app.Server(cfg, handler)
	if err != nil {
		core.LogError("Ошибка создания сервера", map[string]interface{}{"error": err.Error()})
		os.Exit(1)
	}

	// Настраивает обработку сигналов для graceful shutdown
	sigs, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Запускает HTTP-сервер в отдельной горутине
	go func() {
		log.Printf("INFO: http: сервер запущен, addr=%s, env=%s, app=%s", cfg.Addr, cfg.Env, cfg.AppName)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			core.LogError("Ошибка работы сервера", map[string]interface{}{"error": err.Error()})
			os.Exit(1)
		}
	}()

	// Ожидает сигнал завершения и выполняет graceful shutdown
	<-sigs.Done()
	log.Println("INFO: http: начат процесс завершения")
	if err := app.Shutdown(srv, cfg.ShutdownTimeout); err != nil {
		core.LogError("Ошибка завершения работы сервера", map[string]interface{}{"error": err.Error()})
	} else {
		log.Println("INFO: http: завершение работы выполнено")
	}
}

// derive32 генерирует 32-байтовый ключ для CSRF на основе входной строки (OWASP A02)
func derive32(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}
