package app

// internal/app/app.go
import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"os"
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
)

// initTemplates — создаёт шаблоны (было пропущено!)
func initTemplates() (*view.Templates, error) {
	return view.New()
}

// New — главный конструктор Gin
func New(cfg core.Config, db *sqlx.DB, csrfKey []byte) (http.Handler, error) {
	tpl, err := initTemplates() // ← ОПРЕДЕЛЕНО
	if err != nil {
		return nil, err
	}

	r := gin.New()

	r.Use(gin.Logger(), gin.Recovery())

	_ = r.SetTrustedProxies([]string{"127.0.0.1", "::1"})
	r.Use(requestid.New())
	r.Use(RequestTimeout(cfg.RequestTimeout))
	r.Use(withNonceAndDB(db))
	r.Use(core.SecureHeaders())
	if cfg.Secure {
		r.Use(core.HSTS(cfg.Env == "prod"))
	}

	// ← Безопасные сессии
	store := cookie.NewStore(csrfKey)
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: http.SameSiteLaxMode,
	})
	r.Use(sessions.Sessions("mysession", store))

	r.Use(csrf.Middleware(csrf.Options{
		Secret:    string(csrfKey),
		ErrorFunc: csrfError,
	}))

	serveStatic(r)

	registerRoutes(r, tpl)

	return r, nil
}

// RequestTimeout — безопасный таймаут
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
		if ctx.Err() != nil && !c.Writer.Written() {
			c.AbortWithStatusJSON(http.StatusRequestTimeout, gin.H{"error": "timeout"})
		}
	}
}

// withNonceAndDB — nonce + DB в контекст
func withNonceAndDB(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		nonce, err := generateNonce()
		if err != nil {
			core.LogError("nonce error", map[string]interface{}{"error": err})
			core.FailC(c, core.Internal("nonce", err))
			return
		}
		ctx := context.WithValue(c.Request.Context(), core.CtxNonce, nonce)
		ctx = context.WithValue(ctx, storage.CtxDBKey{}, db)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func csrfError(c *gin.Context) {
	core.FailC(c, core.Internal("CSRF invalid", nil))
}

func serveStatic(r *gin.Engine) {
	if _, err := os.Stat("web/assets"); os.IsNotExist(err) {
		core.LogError("web/assets missing", nil)
		return
	}
	r.Static("/assets", "web/assets")
}

func registerRoutes(r *gin.Engine, tpl *view.Templates) {
	r.GET("/", handler.Home(tpl))
	r.GET("/catalog", handler.Catalog(tpl))
	r.GET("/product/:id", handler.Product(tpl))
	r.GET("/form", handler.FormIndex(tpl))
	r.POST("/form", handler.FormSubmit(tpl))
	r.GET("/about", handler.About(tpl))
	r.GET("/debug", handler.Debug)
	r.GET("/catalog/json", handler.CatalogJSON())
	r.NoRoute(handler.NotFound(tpl))

	r.GET("/.well-known/appspecific/com.chrome.devtools.json", func(c *gin.Context) {
		c.Status(http.StatusNoContent) // 204
	})
}

func generateNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
