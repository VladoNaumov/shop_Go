package config

import (
	"log"
	"os"
)

type Config struct {
	AppName  string
	Env      string // dev|staging|prod
	HTTPAddr string // ":8080"
}

func Load() Config {
	cfg := Config{
		AppName:  getEnv("APP_NAME", "shop"),
		Env:      getEnv("APP_ENV", "dev"),
		HTTPAddr: getEnv("HTTP_ADDR", ":8080"),
	}
	// минимальная валидация
	if cfg.HTTPAddr == "" {
		log.Fatal("HTTP_ADDR is required")
	}
	return cfg
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
