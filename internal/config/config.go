package config

import (
	"os"
)

type Config struct {
	AppName string
	Addr    string
	Env     string // "dev" | "prod"
	CSRFKey string
	Secure  bool // prod => true
}

func Load() Config {
	env := getEnv("APP_ENV", "dev")
	return Config{
		AppName: getEnv("APP_NAME", "myApp"),
		Addr:    getEnv("HTTP_ADDR", ":8080"),
		Env:     env,
		CSRFKey: getEnv("CSRF_KEY", "dev-dev-key-change-me-please"),
		Secure:  env == "prod",
	}
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
