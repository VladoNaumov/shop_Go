package core

// config.go

import (
	"log"
	"os"
	"strings"
)

// Config — структура с настройками приложения.
type Config struct {
	AppName string
	Addr    string
	Env     string
	CSRFKey string
	Secure  bool
}

// Load читает конфигурацию из переменных окружения (или задаёт дефолты).
func Load() Config {
	cfg := Config{
		AppName: getEnv("APP_NAME", "myApp"),
		Addr:    getEnv("HTTP_ADDR", ":8080"),
		Env:     getEnv("APP_ENV", "dev"),
		CSRFKey: getEnv("CSRF_KEY", "dev-dev-key-change-me-please"),
		Secure:  getEnv("SECURE", "") == "true",
	}

	// --- Безопасная проверка ключа в продакшене ---
	if cfg.Env == "prod" {
		if cfg.CSRFKey == "dev-dev-key-change-me-please" || len(cfg.CSRFKey) < 8 {
			log.Fatal("CONFIG ERROR: invalid CSRF_KEY in production")
		}
	}

	return cfg
}

// getEnv возвращает переменную окружения или значение по умолчанию.
func getEnv(key, def string) string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return def
	}
	return val
}
