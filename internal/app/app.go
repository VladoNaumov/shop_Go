package app

// app.go
import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"myApp/internal/http/handler"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"

	"myApp/internal/core"
	"myApp/internal/storage"
	"myApp/internal/view"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	csrf "github.com/utrack/gin-csrf"
)

// New — собирает Gin-движок из модулей (middleware, DB, маршруты, шаблоны)
func New(cfg core.Config, db *sqlx.DB, csrfKey []byte) (http.Handler, error) {
	// 1) Шаблоны
	tpl, err := initTemplates()
	if err != nil {
		return nil, err
	}

	// 2) Базовый движок
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// 3) Trusted proxies
	if err := r.SetTrustedProxies([]string{"127.0.0.1", "::1"}); err != nil {
		return nil, err
	}

	// 4) Request ID
	r.Use(requestid.New())

	// 5) Таймаут
	r.Use(RequestTimeout(cfg.RequestTimeout))

	// 6) nonce + DB в контекст
	r.Use(withNonceAndDB(db))

	// 7) Security заголовки
	r.Use(core.SecureHeaders())
	if cfg.Secure {
		r.Use(core.HSTS(cfg.Env == "prod"))
	}

	// 8) СЕССИИ (ОБЯЗАТЕЛЬНО до csrf)
	store := cookie.NewStore(csrfKey) // ключ из derive32(cfg.CSRFKey) ок, длина 32 байта
	r.Use(sessions.Sessions("mysession", store))

	// 9) CSRF (использует sessions.Default(c)
	r.Use(csrf.Middleware(csrf.Options{
		Secret:    string(csrfKey),
		ErrorFunc: csrfError,
	}))

	// 10) Статика
	serveStatic(r)

	// 11) Маршруты
	registerRoutes(r, tpl)

	// Возвращаем как http.Handler (совместимо с твоим main.go)
	return r, nil
}

// initTemplates — создаёт экземпляр шаблонизатора
func initTemplates() (*view.Templates, error) {
	return view.New()
}

// RequestTimeout — простой middleware таймаута для Gin.
// Без внешних зависимостей; дедлайн доступен в c.Request.Context().
func RequestTimeout(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if d <= 0 {
			c.Next()
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		done := make(chan struct{})
		go func() {
			c.Next()
			close(done)
		}()

		select {
		case <-done:
			return
		case <-ctx.Done():
			if !c.Writer.Written() {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
					"error":  "request timeout",
					"status": http.StatusServiceUnavailable,
				})
			} else {
				c.Abort()
			}
			return
		}
	}
}

// withNonceAndDB — кладём nonce в Gin-контекст И в request.Context,
// плюс пробрасываем *sqlx.DB в request.Context (как было в chi-версии).
func withNonceAndDB(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		nonce, err := generateNonce()
		if err != nil {
			core.LogError("Ошибка генерации nonce", map[string]interface{}{"error": err.Error()})
			core.FailC(c, core.Internal("Ошибка генерации nonce", err))
			return
		}

		// Сохраняем в Gin-контекст:
		c.Set("nonce", nonce)

		// И в стандартный context, чтобы старые net/http-хэндлеры тоже видели:
		ctx := context.WithValue(c.Request.Context(), core.CtxNonce, nonce)
		ctx = context.WithValue(ctx, storage.CtxDBKey{}, db)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// csrfError — единообразный ответ при ошибке CSRF
func csrfError(c *gin.Context) {
	core.FailC(c, core.Internal("CSRF токен недействителен или отсутствует", nil))
}

// serveStatic — статика из web/assets
func serveStatic(r *gin.Engine) {
	if _, err := os.Stat("web/assets"); os.IsNotExist(err) {
		core.LogError("Директория web/assets не найдена", nil)
		return
	}
	// /assets/* -> ./web/assets
	r.Static("/assets", "web/assets")
}

// registerRoutes — регистрируем маршруты приложения
// Если твои handlers имеют сигнатуру net/http (func(w,r)), можно оборачивать через gin.WrapF/WrapH.
func registerRoutes(r *gin.Engine, tpl *view.Templates) {
	// Страницы (все принимают *gin.Context)
	r.GET("/", handler.Home(tpl))
	r.GET("/catalog", handler.Catalog(tpl))
	r.GET("/product/:id", handler.Product(tpl))
	r.GET("/form", handler.FormIndex(tpl))
	r.POST("/form", handler.FormSubmit(tpl))
	r.GET("/about", handler.About(tpl))

	// Отладка (JSON)
	r.GET("/debug", handler.Debug)

	// JSON-эндпоинт каталога
	r.GET("/catalog/json", handler.CatalogJSON())

	// 404
	r.NoRoute(handler.NotFound(tpl))
}

// generateNonce — криптографически стойкий nonce для CSP (base64)
func generateNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
