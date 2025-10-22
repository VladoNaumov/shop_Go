package core

// config.go

import (
	"crypto/rand"
	"encoding/base64"
	"os"
	"strings"
	"time"
)

// Config — настройки приложения
type Config struct {
	AppName           string        // Имя приложения
	Addr              string        // Адрес HTTP-сервера (например, ":8080")
	Env               string        // Среда выполнения (dev, prod)
	CSRFKey           string        // Ключ для CSRF-защиты (строка, предпочтительно base64 от 32 байт)
	Secure            bool          // Включает HTTPS-поведение приложения (cookie Secure, HSTS и т.п.)
	TLSOffloaded      bool          // true если TLS завершается на прокси (nginx), тогда cert/key не требуются
	CertFile          string        // Путь к TLS-сертификату (только если TLS не offloaded)
	KeyFile           string        // Путь к TLS-ключу (только если TLS не offloaded)
	ShutdownTimeout   time.Duration // Таймаут для graceful shutdown
	ReadHeaderTimeout time.Duration // Таймаут чтения заголовков HTTP-запроса
	ReadTimeout       time.Duration // Таймаут чтения HTTP-запроса
	WriteTimeout      time.Duration // Таймаут записи HTTP-ответа
	IdleTimeout       time.Duration // Таймаут простоя соединения
	RequestTimeout    time.Duration // Таймаут обработки запроса в middleware
}

// Load загружает конфигурацию из ENV с дефолтами
func Load() Config {
	cfg := Config{
		AppName:           getEnv("APP_NAME", "myApp"),
		Addr:              getEnv("HTTP_ADDR", ":8080"),
		Env:               getEnv("APP_ENV", "dev"),
		CSRFKey:           getEnv("CSRF_KEY", generateRandomKey()),
		Secure:            getEnvBool("SECURE", false),
		TLSOffloaded:      getEnvBool("TLS_OFFLOADED", false), // если true — TLS у nginx
		CertFile:          getEnv("TLS_CERT_FILE", ""),
		KeyFile:           getEnv("TLS_KEY_FILE", ""),
		ShutdownTimeout:   getEnvDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		ReadHeaderTimeout: getEnvDuration("READ_HEADER_TIMEOUT", 5*time.Second),
		ReadTimeout:       getEnvDuration("READ_TIMEOUT", 10*time.Second),
		WriteTimeout:      getEnvDuration("WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:       getEnvDuration("IDLE_TIMEOUT", 60*time.Second),
		RequestTimeout:    getEnvDuration("REQUEST_TIMEOUT", 15*time.Second),
	}

	// Валидация для prod
	if strings.ToLower(cfg.Env) == "prod" {
		// CSRF key должен быть криптографически сильным — минимум 32 байта.
		if !isKeyStrong(cfg.CSRFKey, 32) {
			LogError("Недостаточная длина CSRF_KEY в продакшене (требуется минимум 32 байта)", map[string]interface{}{"provided_length": len(cfg.CSRFKey)})
			os.Exit(1)
		}

		// Адрес обязателен в проде
		if cfg.Addr == "" {
			LogError("Отсутствует HTTP_ADDR в продакшене", nil)
			os.Exit(1)
		}

		// Приложение в проде должно знать о том, что трафик защищён — Secure должен быть true.
		// Даже если TLS offloaded на прокси — приложение должно вести себя как secure (cookie Secure, HSTS и т.д.)
		if !cfg.Secure {
			LogError("SECURE должен быть true в продакшене — приложение должно работать в HTTPS-режиме (даже при offload на прокси)", nil)
			os.Exit(1)
		}

		// Если TLS не offloaded на прокси — требуем наличия cert/key
		if !cfg.TLSOffloaded && (cfg.CertFile == "" || cfg.KeyFile == "") {
			LogError("TLS_CERT_FILE / TLS_KEY_FILE отсутствуют в продакшене, а TLS не offloaded", nil)
			os.Exit(1)
		}
	}

	return cfg
}

// getEnv — строка из ENV или дефолт
func getEnv(key, def string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return def
}

// getEnvBool — bool из ENV (поддержка true/1/yes/on)
func getEnvBool(key string, def bool) bool {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return def
	}
	v := strings.ToLower(val)
	return v == "true" || v == "1" || v == "yes" || v == "on"
}

// getEnvDuration — time.Duration из ENV.
// Поддерживает форматы: "30s", "1m", а также просто число (в секундах).
func getEnvDuration(key string, def time.Duration) time.Duration {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return def
	}

	// Попытка распарсить стандартную длительность
	if d, err := time.ParseDuration(val); err == nil {
		return d
	}

	// Если пользователь передал просто число — интерпретируем как секунды
	if secs, err := time.ParseDuration(val + "s"); err == nil {
		return secs
	}

	LogError("Неверный формат длительности", map[string]interface{}{"key": key, "value": val})
	return def
}

// generateRandomKey — криптостойкий ключ 32 байта → base64
func generateRandomKey() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		LogError("Ошибка генерации CSRF-ключа", map[string]interface{}{"error": err.Error()})
		panic("невозможно сгенерировать ключ")
	}
	return base64.StdEncoding.EncodeToString(b)
}

// isKeyStrong проверяет, что ключ содержит минимум minBytes байт.
// Если строка выглядит как base64 — пытаемся декодировать и проверяем декодированные байты.
// Иначе проверяем длину строки (в байтах).
func isKeyStrong(key string, minBytes int) bool {
	if key == "" {
		return false
	}

	// Попытка base64-декодирования (если получилось — смотрим длину декодированных данных)
	if decoded, err := base64.StdEncoding.DecodeString(key); err == nil {
		return len(decoded) >= minBytes
	}

	// Иначе проверим raw-байтовую длину строки
	return len([]byte(key)) >= minBytes
}
