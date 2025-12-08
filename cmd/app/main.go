package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
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

// Константы для ключей Gin Context (для доступа к ресурсам из обработчиков)
const (
	ContextDBKey    = "db_connection"
	ContextNonceKey = "csp_nonce"
)

// Main — вход: конфиг, инициализация и запуск.
func main() {
	cfg := core.Load()

	core.LogInfo("Приложение запущено", map[string]interface{}{
		"env":    cfg.Env,
		"addr":   cfg.Addr,
		"secure": cfg.Secure,
		"app":    cfg.AppName,
	})

	core.InitDailyLog()

	if err := run(&cfg); err != nil {
		core.LogError("Критическая ошибка запуска", map[string]interface{}{"error": err})
		os.Exit(1)
	}
}

// run — основная функция lifecycle: storage, app, сервер с graceful shutdown.
func run(cfg *core.Config) error {
	db, err := initStorage()
	if err != nil {
		return err
	}
	defer func() {
		if err := storage.Close(db); err != nil {
			core.LogError("Ошибка закрытия БД", map[string]interface{}{"error": err})
		}
		core.Close()
	}()

	csrfKey := deriveSecureKey(cfg.CSRFKey)

	appHandler, err := newApp(*cfg, db, csrfKey)
	if err != nil {
		return err
	}

	srv := newHTTPServer(*cfg, appHandler)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	core.LogInfo("Сервер запущен, ждём сигнал завершения...", map[string]interface{}{"addr": cfg.Addr})
	go runServer(srv)

	<-ctx.Done()
	core.LogInfo("Завершение...", nil)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}

	return nil
}

// initStorage — инициализация БД и миграций
func initStorage() (*sqlx.DB, error) {
	db, err := storage.NewDB()
	if err != nil {
		return nil, err
	}
	migrations := storage.NewMigrations(db)
	if err := migrations.RunMigrations(); err != nil {
		return nil, err
	}
	return db, nil
}

// newApp — Главный конструктор Gin, собирает всю цепочку middleware и роуты.
func newApp(cfg core.Config, db *sqlx.DB, csrfKey []byte) (http.Handler, error) {
	tpl, err := initTemplates()
	if err != nil {
		return nil, err
	}

	r := gin.New()

	if strings.ToLower(cfg.Env) == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	r.Use(gin.Logger(), gin.Recovery())

	_ = r.SetTrustedProxies([]string{"127.0.0.1", "::1"})

	// Корреляция запросов (RequestID)
	r.Use(requestid.New())

	// Таймаут запроса (отсекаем "висящие" клиенты)
	r.Use(RequestTimeout(cfg.RequestTimeout))

	// ⭐ BEST PRACTICE: Кладём nonce и DB в Gin Context.
	// Это должно идти до middleware, которое их использует (CSP, обработчики).
	r.Use(withNonceAndDB(db))

	// Security заголовки (X-Frame-Options, X-Content-Type-Options и пр.)
	r.Use(core.SecureHeaders())

	// CSP (Content-Security-Policy) — читаем nonce из Gin Context (ContextNonceKey)
	// Используем локальную реализацию CSPBasic(), которая читает nonce по ключу ContextNonceKey.
	r.Use(CSPBasic())

	// Безопасные cookie-сессии
	store := cookie.NewStore(csrfKey)
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: http.SameSiteLaxMode,
	})
	r.Use(sessions.Sessions("mysession", store))

	// CSRF защита форм.
	r.Use(csrf.Middleware(csrf.Options{
		Secret:    base64.StdEncoding.EncodeToString(csrfKey),
		ErrorFunc: csrfError,
	}))

	// Статика
	serveStatic(r, cfg.Env)

	// Роуты
	registerRoutes(r, tpl)

	return r, nil
}

// RequestTimeout — безопасный таймаут для всего запроса.
// ⭐ Улучшение: Использование Context.Done() для проверки таймаута Gin-стиле.
func RequestTimeout(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if d <= 0 {
			c.Next()
			return
		}

		// Создаем контекст таймаута на основе контекста запроса
		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()

		// Заменяем контекст запроса на новый с таймаутом
		c.Request = c.Request.WithContext(ctx)

		// Выполняем цепочку middleware/обработчиков
		c.Next()

		// Если контекст протух И ответ еще не был отправлен:
		if err := ctx.Err(); err != nil && errors.Is(err, context.DeadlineExceeded) {
			core.LogError("Запрос завершился по таймауту", map[string]interface{}{
				"timeout": d.String(),
				"path":    c.FullPath(),
				"error":   err.Error(),
			})
			// Аборт и ответ с 408 (если ответ ещё не был отправлен)
			if !c.Writer.Written() {
				c.AbortWithStatusJSON(http.StatusRequestTimeout, gin.H{"error": "request timeout"})
			}
		}
	}
}

// withNonceAndDB — генерирует CSP nonce и кладёт вместе с *sqlx.DB в Gin Context.
// ⭐ BEST PRACTICE: Используем c.Set() для передачи данных в Gin Context.
func withNonceAndDB(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		nonce, err := generateNonce()
		if err != nil {
			core.LogError("Ошибка генерации CSP nonce", map[string]interface{}{"error": err})
			core.FailC(c, core.Internal("nonce generation failed", err))
			return
		}

		// ⭐ Установка в Gin Context с использованием строковых ключей.
		c.Set(ContextNonceKey, nonce)
		c.Set(ContextDBKey, db)

		c.Next()
	}
}

// CSPBasic — простая (но безопасная) CSP-политика, которая читает nonce из Gin Context.
// Берёт nonce по ключу ContextNonceKey; если nonce отсутствует, ставит политику без nonce.
func CSPBasic() gin.HandlerFunc {
	return func(c *gin.Context) {
		var nonce string
		if v, ok := c.Get(ContextNonceKey); ok {
			if s, _ := v.(string); s != "" {
				nonce = s
			}
		}

		// Построим базовую политику. Подстраивайте по нуждам приложения.
		// Включаем nonce для inline-скриптов, если он есть.
		policyBuilder := []string{
			"default-src 'self'",
			"object-src 'none'",
			"base-uri 'self'",
			"frame-ancestors 'none'",
		}

		if nonce != "" {
			// script-src с nonce
			policyBuilder = append(policyBuilder, fmt.Sprintf("script-src 'self' 'nonce-%s'", nonce))
			// style-src — лучше не полагаться на nonce для стилей, если используете inline styles, можно добавить 'unsafe-inline' или использовать nonce так же.
			policyBuilder = append(policyBuilder, "style-src 'self' 'unsafe-inline'")
		} else {
			// Если нет nonce — запретим inline-скрипты, но разрешаем self
			policyBuilder = append(policyBuilder, "script-src 'self'")
			policyBuilder = append(policyBuilder, "style-src 'self' 'unsafe-inline'")
		}

		// Собираем политику и ставим заголовок
		policy := strings.Join(policyBuilder, "; ")
		c.Header("Content-Security-Policy", policy)

		c.Next()
	}
}

// csrfError — единообразный ответ на невалидный CSRF-токен (HTTP 403 Forbidden).
func csrfError(c *gin.Context) {
	core.FailC(c, core.Forbidden("CSRF token is invalid or missing."))
}

// serveStatic — раздача файлов из web/assets. Отключает кэш в режиме dev.
func serveStatic(r *gin.Engine, env string) {
	if _, err := os.Stat("web/assets"); os.IsNotExist(err) {
		core.LogError("Папка web/assets отсутствует, статика не будет доступна", nil)
		return
	}

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

// newHTTPServer — создаёт http. Server с параметрами из конфига
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

// runServer — запускает сервер и логирует падения
func runServer(srv *http.Server) {
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		core.LogError("Сервер упал", map[string]interface{}{"error": err})
		os.Exit(1)
	}
}

// deriveSecureKey — генерирует 32-байтовый криптографически стойкий ключ для CSRF.
func deriveSecureKey(secret string) []byte {
	if len(secret) == 0 {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			panic("unable to generate random bytes: " + err.Error())
		}
		return b
	}

	// Растягивание ключа через PBKDF2 (4096 итераций)
	salt := []byte("myapp-session-salt")
	return pbkdf2.Key([]byte(secret), salt, 4096, 32, sha256.New)
}
