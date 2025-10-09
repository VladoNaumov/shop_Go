package main

// main.go

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

	"myApp/internal/core"

	"github.com/gorilla/csrf"
)

func main() {
	// --- 1. Загружаем конфиг ---
	cfg := core.Load()

	// --- 2. Настраиваем лог-файл (по дате) ---
	// Создаст logs/DD-MM-YYYY.log и перенаправит стандартный лог туда.
	core.InitDailyLog()

	// 🔁 --- 3. Автоматическая ротация логов каждый день в полночь ---
	go func() {
		for {
			next := time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour)
			time.Sleep(time.Until(next))
			core.InitDailyLog() // переключаемся на новый файл logs/DD-MM-YYYY.log
		}
	}()

	// --- 4. Санити-проверки для prod ---
	if cfg.Env == "prod" {
		if cfg.CSRFKey == "" {
			log.Println("ERROR: missing CSRF_KEY in prod")
			os.Exit(1) // фаталим прод без ключа
		}
		if !cfg.Secure {
			log.Println("WARN: APP_ENV=prod but Secure=false; HTTPS/HSTS disabled")
		}
	}

	// --- 5. Создаём маршрутизатор (роутер) ---
	router := core.NewRouter() // все обработчики и маршруты внутри пакета internal/http

	// --- 6. Оборачиваем роутер дополнительными мидлварами ---
	var h http.Handler = router

	// В продакшене включаем HSTS (строгая политика HTTPS)
	if cfg.Secure {
		h = hsts(h)
	}

	// --- 7. Настраиваем CSRF-защиту ---
	// Gorilla CSRF требует 32-байтовый ключ, берём SHA256 от строки из конфига.
	csrfKey := derive32(cfg.CSRFKey)
	h = csrf.Protect(
		csrfKey,
		csrf.Secure(cfg.Secure),             // только через HTTPS, если Secure = true
		csrf.SameSite(csrf.SameSiteLaxMode), // безопасный и удобный режим для форм
		csrf.HttpOnly(true),                 // токен не будет доступен из JS
		csrf.Path("/"),                      // CSRF-токен действует на весь сайт
	)(h)

	// --- 8. Создаём HTTP-сервер ---
	srv := core.Server(cfg.Addr, h)

	// --- 9. Настраиваем "graceful shutdown" (мягкое завершение) ---
	// Перехватываем сигналы ОС: Ctrl+C или SIGTERM (от Docker/сервиса)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// --- 10. Запускаем сервер в отдельной горутине ---
	go func() {
		log.Printf("INFO: http: listening addr=%s env=%s app=%s", cfg.Addr, cfg.Env, cfg.AppName)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			// Ошибка запуска сервера — логируем и завершаем процесс
			log.Printf("ERROR: http: server error: %v", err)
			os.Exit(1)
		}
	}()

	// --- 11. Ждём сигнал завершения ---
	<-ctx.Done()
	log.Println("INFO: http: shutdown started")

	// --- 12. Завершаем сервер с таймаутом 10 секунд ---
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("ERROR: http: shutdown error: %v", err)
	} else {
		log.Println("INFO: http: shutdown complete")
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
