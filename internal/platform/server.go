package platform

// Создаёт http.Server с безопасными таймаутами/лимитами — защита от Slowloris/DoS на уровне соединений.

import (
	"net/http"
	"time"
)

func Server(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}
}
