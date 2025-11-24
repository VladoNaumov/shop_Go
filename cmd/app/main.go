// main.go — точка входа приложения myApp. Содержит всю логику инициализации, middleware, роутов и настроек.

package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"myApp/internal/core"
	"myApp/internal/http/handler"
	"myApp/internal/storage"
	"myApp/internal/view"

	"github.com/gin-contrib/requestid"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	csrf "github.com/utrack/gin-csrf"
	"golang.org/x/crypto/pbkdf2"
)

// ВАЖНО: CSRF secret (долгоживущий ключ) ≠ CSP nonce (случайное значение на КАЖДЫЙ запрос).

func main() {
	// Загружаем конфиг (из .env, переменных окружения или файла)
	cfg := core.Load()

	// Логируем старт с параметрами окружения
	core.LogInfo("Приложение запущено", map[string]interface{}{
		"env":    cfg.Env,    // режим: dev / prod
		"addr":   cfg.Addr,   // адрес HTTP-сервера
		"secure": cfg.Secure, // HTTPS включён или нет
		"app":    cfg.AppName,
	})

	// Инициализируем ежедневный лог-файл (по дате)
	core.InitDailyLog()

	// Подключаем базу данных (sqlx.DB)
	db, err := storage.NewDB()
	if err != nil {
		core.LogError("Ошибка БД", map[string]interface{}{"error": err})
		os.Exit(1)
	}

	// Запускаем миграции, если есть (обновление структуры БД)
	migrations := storage.NewMigrations(db)
	if err := migrations.RunMigrations(); err != nil {
		core.LogError("Ошибка миграций", map[string]interface{}{"error": err})
		os.Exit(1)
	}

	// Генерируем или производим derivation CSRF-ключа (32 байта)
	// Используется для защиты форм и сессий
	csrfKey := deriveSecureKey(cfg.CSRFKey)

	// Инициализируем приложение (Gin, middleware, routes, CSP nonce, CSRF-защиту, Раздаёт статику /assets из web/assets)
	appHandler, err := newApp(cfg, db, csrfKey)
	if err != nil {
		core.LogError("Ошибка app.New", map[string]interface{}{"error": err})
		os.Exit(1)
	}

	// Создаём HTTP-сервер с таймаутами
	srv := newHTTPServer(cfg, appHandler)

	// Создаём контекст, который будет отменён при сигнале SIGINT/SIGTERM
	// (нужно для graceful shutdown)
	sigs, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Запускаем сервер в отдельной горутине
	go runServer(srv, cfg)

	// Ждём сигнал завершения (Ctrl+C или systemd stop)
	<-sigs.Done()
	core.LogInfo("Завершение...", nil)

	// Плавно останавливаем сервер
	if err := srv.Shutdown(context.Background()); err != nil {
		core.LogError("Ошибка shutdown", map[string]interface{}{"error": err})
	}

	// Закрываем соединение с БД
	if err := storage.Close(db); err != nil {
		core.LogError("Ошибка закрытия БД", map[string]interface{}{"error": err})
	}

	// Закрываем логи, если нужно
	core.Close()
}

// newApp — Главный конструктор Gin, собирает всю цепочку middleware и роуты.
// (Ранее был в internal/app/app.go; инлайнен для одного файла)
func newApp(cfg core.Config, db *sqlx.DB, csrfKey []byte) (http.Handler, error) {
	// initTemplates — Инициализация шаблонов.
	tpl, err := initTemplates()
	if err != nil {
		return nil, err
	}

	r := gin.New()

	// Настройка режима Gin (ReleaseMode в Prod)
	if strings.ToLower(cfg.Env) == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Базовые логи/восстановление паник
	r.Use(gin.Logger(), gin.Recovery())

	// Trust proxy только локально (добавь IP внешних прокси, если нужно)
	_ = r.SetTrustedProxies([]string{"127.0.0.1", "::1"})

	// Корреляция запросов (RequestID)
	r.Use(requestid.New())

	// Таймаут запроса (отсекаем "висящие" клиенты)
	r.Use(RequestTimeout(cfg.RequestTimeout))

	// Кладём nonce и DB в контекст запроса — это нужно ДО установки CSP.
	r.Use(withNonceAndDB(db))

	// Security заголовки (X-Frame-Options, X-Content-Type-Options и пр.)
	r.Use(core.SecureHeaders())

	// CSP (Content-Security-Policy)
	r.Use(core.CSPBasic())

	// Безопасные cookie-сессии (HttpOnly, SameSite, Secure=prod)
	store := cookie.NewStore(csrfKey)
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		Secure:   cfg.Secure, // Используем cfg.Secure для автоматического переключения
		SameSite: http.SameSiteLaxMode,
	})
	r.Use(sessions.Sessions("mysession", store))

	// CSRF защита форм. Использует сессию.
	r.Use(csrf.Middleware(csrf.Options{
		Secret:    base64.StdEncoding.EncodeToString(csrfKey), // Base64 для безопасности (избежать nul-байт)
		ErrorFunc: csrfError,                                  // Использует 403 Forbidden через core.FailC
	}))

	// Статика (с условным отключением кэша в Dev-режиме)
	serveStatic(r, cfg.Env)

	// Роуты
	registerRoutes(r, tpl)

	return r, nil
}

// RequestTimeout — безопасный таймаут для всего запроса.
// Если истек таймаут и ответ еще не был отправлен, возвращает 408 Request Timeout.
func RequestTimeout(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if d <= 0 {
			c.Next()
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()

		// Если контекст протух и ещё ничего не было записано в ответ — вернём 408
		if ctx.Err() != nil && !c.Writer.Written() {
			// Логируем факт таймаута для мониторинга
			core.LogError("Запрос завершился по таймауту", map[string]interface{}{
				"timeout": d.String(),
				"path":    c.FullPath(),
			})
			c.AbortWithStatusJSON(http.StatusRequestTimeout, gin.H{"error": "request timeout"})
		}
	}
}

// withNonceAndDB — генерирует CSP nonce и кладёт вместе с *sqlx.DB в контекст запроса.
func withNonceAndDB(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		nonce, err := generateNonce()
		if err != nil {
			// Если nonce не сгенерировался, это критическая внутренняя ошибка
			core.LogError("Ошибка генерации CSP nonce", map[string]interface{}{"error": err})
			core.FailC(c, core.Internal("nonce generation failed", err))
			return
		}

		// Кладём nonce в контекст с использованием ключа CtxNonce из core
		ctx := context.WithValue(c.Request.Context(), core.CtxNonce, nonce)

		// Кладём DB в контекст с использованием ключа CtxDBKey из storage (предполагается его определение)
		ctx = context.WithValue(ctx, storage.CtxDBKey{}, db)

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// csrfError — единообразный ответ на невалидный CSRF-токен (HTTP 403 Forbidden).
func csrfError(c *gin.Context) {
	// 403 Forbidden более точен, чем 500 Internal
	core.FailC(c, core.Forbidden("CSRF token is invalid or missing."))
}

// serveStatic — раздача файлов из web/assets. Отключает кэш в режиме dev.
func serveStatic(r *gin.Engine, env string) {
	if _, err := os.Stat("web/assets"); os.IsNotExist(err) {
		core.LogError("Папка web/assets отсутствует, статика не будет доступна", nil)
		return
	}

	// В режиме dev — отключаем кэш для моментального обновления в браузере
	if strings.ToLower(env) == "dev" {
		r.Use(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/assets/") {
				c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
				c.Header("Pragma", "no-cache")
				c.Header("Expires", "0")
			}
			c.Next()
		})
	}
	// TODO: В prod стоит добавить кэширование (например, max-age=31536000)

	// Раздача статики
	r.Static("/assets", "web/assets")
}

// registerRoutes — Регистрация всех маршрутов приложения.
func registerRoutes(r *gin.Engine, tpl *view.Templates) {
	r.GET("/", handler.Home(tpl))
	r.GET("/catalog", handler.Catalog(tpl))
	r.GET("/product/:id", handler.Product(tpl))
	r.GET("/form", handler.FormIndex(tpl))
	r.POST("/form", handler.FormSubmit(tpl))
	r.GET("/about", handler.About(tpl))
	r.GET("/debug", handler.Debug)
	r.GET("/catalog/json", handler.CatalogJSON())

	// Обработчик 404
	r.NoRoute(handler.NotFound(tpl))
}

// initTemplates — Инициализация шаблонов.
func initTemplates() (*view.Templates, error) {
	return view.New()
}

// generateNonce — Создаёт 16 байт криптографически стойкой случайности и кодирует в Base64.
func generateNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// newHTTPServer — создаёт http.Server с параметрами из конфига
func newHTTPServer(cfg core.Config, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              cfg.Addr,              // адрес (например ":8080")
		Handler:           h,                     // обработчик (Gin engine)
		ReadHeaderTimeout: cfg.ReadHeaderTimeout, // таймаут заголовков
		ReadTimeout:       cfg.ReadTimeout,       // общий таймаут чтения
		WriteTimeout:      cfg.WriteTimeout,      // таймаут записи
		IdleTimeout:       cfg.IdleTimeout,       // таймаут keep-alive
	}
}

// runServer — запускает сервер и логирует падения
func runServer(srv *http.Server, cfg core.Config) {
	core.LogInfo("Сервер запущен", map[string]interface{}{"addr": cfg.Addr})

	// Запускаем HTTP-сервер
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		// Если ошибка не "сервер закрыт" — это краш
		core.LogError("Сервер упал", map[string]interface{}{"error": err})
		os.Exit(1)
	}
}

// deriveSecureKey — генерирует 32-байтовый криптографически стойкий ключ для CSRF, если secret пустой — создаёт новый.
func deriveSecureKey(secret string) []byte {
	if len(secret) == 0 {
		// Если в конфиге нет ключа — генерируем случайный
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			panic("unable to generate random bytes: " + err.Error())
		}
		return b
	}

	// Если ключ задан, "растягиваем" его через PBKDF2
	// — безопасный способ получить ключ фиксированной длины
	salt := []byte("myapp-session-salt")
	return pbkdf2.Key([]byte(secret), salt, 4096, 32, sha256.New)
}
