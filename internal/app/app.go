package app

// internal/app/app.go — Главный конструктор Gin-приложения.
// Здесь собираются все middleware, роуты и настройки безопасности.

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"myApp/internal/core"
	"myApp/internal/http/handler"
	"myApp/internal/storage"
	"myApp/internal/view"

	"github.com/gin-contrib/requestid"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
	csrf "github.com/utrack/gin-csrf"
)

// ВАЖНО: CSRF secret (долгоживущий ключ) ≠ CSP nonce (случайное значение на КАЖДЫЙ запрос).

// initTemplates — Инициализация шаблонов.
func initTemplates() (*view.Templates, error) {
	return view.New()
}

// New — Главный конструктор Gin, собирает всю цепочку middleware и роуты.
func New(cfg core.Config, db *sqlx.DB, csrfKey []byte) (http.Handler, error) {
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
		Secret:    string(csrfKey),
		ErrorFunc: csrfError, // Использует 403 Forbidden через core.FailC
	}))

	// Статика (с условным отключением кэша в Dev-режиме)
	serveStatic(r, cfg.Env)

	// Роуты
	registerRoutes(r, tpl, cfg, db)

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

// JWTMiddleware — Проверяет JWT в заголовке Authorization (Bearer).
func JWTMiddleware(cfg core.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			core.FailC(c, core.Unauthorized("Authorization header missing"))
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			core.FailC(c, core.Unauthorized("Invalid Authorization header format"))
			return
		}

		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {

				core.LogError("Unexpected signing method in JWT", map[string]interface{}{
					"algorithm": token.Header["alg"],
					"jwt_token": parts[1], // Логируем сам токен для отладки, если нужно
					"source":    "JWTMiddleware",
				})
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(cfg.JWT.Secret), nil
		})

		if err != nil || !token.Valid {
			core.FailC(c, core.Unauthorized("Invalid or expired JWT"))
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			core.FailC(c, core.Internal("Failed to parse JWT claims", nil))
			return
		}
		ctx := context.WithValue(c.Request.Context(), core.CtxUser, claims)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// LoginRequest — Структура для валидации тела запроса на логин.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginHandler — Обработчик логина, выдает JWT.
// Принимает конфигурацию (cfg) и подключение к БД (db) и возвращает обработчик gin.
func LoginHandler(cfg core.Config, db *sqlx.DB) gin.HandlerFunc {
	// Возвращаем анонимную функцию, которая будет обработчиком Gin.
	return func(c *gin.Context) {
		// Объявляем структуру для хранения данных запроса (логин/пароль).
		var req LoginRequest

		// 1. ПАРСИНГ И ВАЛИДАЦИЯ ВХОДНЫХ ДАННЫХ
		// Пытаемся привязать JSON-тело запроса к структуре `req`.
		if err := c.ShouldBindJSON(&req); err != nil {
			// Если произошла ошибка парсинга (например, неверный JSON формат) или
			// валидации (если в LoginRequest есть теги "binding"),
			// отправляем клиенту ошибку 400 Bad Request.
			core.FailC(c, core.BadRequest("Invalid request body", err))
			return
		}

		// 2. АУТЕНТИФИКАЦИЯ ПОЛЬЗОВАТЕЛЯ
		// TODO: Заглушка: замените на проверку в БД.
		// В текущей реализации: жестко закодированная проверка логина/пароля.
		// В реальном приложении здесь должен быть запрос к БД,
		// сравнение хеша пароля, и получение данных пользователя.
		if req.Username != "test" || req.Password != "test123" {
			// Если учетные данные не совпадают, отправляем ошибку 401 Unauthorized.
			core.FailC(c, core.Unauthorized("Invalid credentials"))
			return
		}

		// 3. СОЗДАНИЕ PAYLOAD (CLAIMS) ДЛЯ JWT
		// Создаем набор утверждений (claims), которые будут храниться в токене.
		claims := jwt.MapClaims{
			// "sub" (Subject): Идентификатор пользователя (здесь — имя).
			// Это основное "что" идентифицирует пользователя.
			"sub": req.Username,

			// "exp" (Expiration Time): Время истечения срока действия токена (Unix-время).
			// Токен будет недействителен после этого времени.
			"exp": time.Now().Add(cfg.JWT.Expiration).Unix(),

			// "iat" (Issued At): Время выдачи токена (Unix-время).
			"iat": time.Now().Unix(),
		}

		// 4. ПОДПИСАНИЕ ТОКЕНА
		// Создаем новый токен, используя выбранный алгоритм (HS256) и утверждения (claims).
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

		// Подписываем токен с помощью секретного ключа (`cfg.JWT.Secret`).
		// Результатом является строка — сам JWT в формате header.payload.signature.
		tokenString, err := token.SignedString([]byte(cfg.JWT.Secret))

		if err != nil {
			// Если не удалось подписать токен (редко, обычно проблема конфигурации),
			// отправляем ошибку 500 Internal Server Error.
			core.FailC(c, core.Internal("Failed to generate JWT", err))
			return
		}

		// 5. ОТПРАВКА ОТВЕТА
		// Отправляем успешный ответ (HTTP 200 OK), содержащий сгенерированный JWT.
		c.JSON(http.StatusOK, gin.H{
			"token": tokenString, // Клиент будет использовать этот токен для доступа к защищенным маршрутам.
		})
	}
}

// registerRoutes — Регистрация всех маршрутов приложения.
func registerRoutes(r *gin.Engine, tpl *view.Templates, cfg core.Config, db *sqlx.DB) {
	// Группы роутов и прочие обработчики
	r.GET("/", handler.Home(tpl))
	r.GET("/catalog", handler.Catalog(tpl))
	r.GET("/product/:id", handler.Product(tpl))
	r.GET("/form", handler.FormIndex(tpl))
	r.POST("/form", handler.FormSubmit(tpl))
	r.GET("/about", handler.About(tpl))
	r.GET("/debug", handler.Debug)
	r.GET("/catalog/json", handler.CatalogJSON())

	// Роут для логина
	r.POST("/login", LoginHandler(cfg, db))

	// Защищенные роуты с JWT
	protected := r.Group("/api")
	protected.Use(JWTMiddleware(cfg))
	{
		protected.GET("/user", func(c *gin.Context) {
			claims := c.Request.Context().Value(core.CtxUser).(jwt.MapClaims)
			c.JSON(http.StatusOK, gin.H{
				"user": claims["sub"],
			})
		})
	}

	// Обработчик 404
	r.NoRoute(handler.NotFound(tpl))
}

// generateNonce — Создаёт 16 байт криптографически стойкой случайности и кодирует в Base64.
func generateNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
