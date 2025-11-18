package core

// auth.go - Authorization (Bearer)
import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
)

// LoginRequest — Структура для привязки JSON-тела логин-запроса.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// JWTMiddleware — Проверяет JWT в заголовке Authorization (Bearer).
// Применяется для защиты маршрутов, требующих аутентификации.
func JWTMiddleware(cfg Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// 1. Проверка наличия заголовка
			FailC(c, Unauthorized("Authorization header missing"))
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			// 2. Проверка формата (ожидается "Bearer <token>")
			FailC(c, Unauthorized("Invalid Authorization header format"))
			return
		}

		// 3. Парсинг и валидация токена
		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			// Keyfunc: Проверяем, что используется ожидаемый алгоритм подписи (HS256)
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				// Логируем попытку использования неожиданного алгоритма
				LogError("Unexpected signing method in JWT", map[string]interface{}{
					"algorithm": token.Header["alg"],
					"source":    "JWTMiddleware",
				})
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			// Возвращаем секретный ключ для проверки подписи
			return []byte(cfg.JWT.Secret), nil
		})

		if err != nil || !token.Valid {
			// 4. Ошибка валидации (просрочен, неверная подпись и т.д.)
			FailC(c, Unauthorized("Invalid or expired JWT"))
			return
		}

		// 5. Извлечение утверждений (Claims)
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			// Критическая ошибка: утверждения токена не удалось преобразовать в MapClaims
			FailC(c, Internal("Failed to parse JWT claims", nil))
			return
		}

		// 6. Сохранение утверждений в контексте запроса
		// Это позволяет последующим обработчикам получить данные пользователя (например, sub, user_id)
		ctx := context.WithValue(c.Request.Context(), CtxUser, claims)
		c.Request = c.Request.WithContext(ctx)

		// 7. Продолжение цепочки middleware
		c.Next()
	}
}

// LoginHandler — Обработчик логина, выдает JWT.
func LoginHandler(cfg Config, db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			// Ошибка парсинга или валидации входных данных
			FailC(c, BadRequest("Invalid request body", err))
			return
		}

		// TODO: Заглушка: замените на проверку в БД.
		// В реальном приложении: поиск пользователя по req.Username и проверка хеша пароля.
		if req.Username != "test" || req.Password != "test123" {
			FailC(c, Unauthorized("Invalid credentials"))
			return
		}

		// 1. Создание утверждений (claims)
		claims := jwt.MapClaims{
			"sub": req.Username,
			// Устанавливаем время истечения токена (Expiration)
			"exp": time.Now().Add(cfg.JWT.Expiration).Unix(),
			// Устанавливаем время выдачи токена (Issued At)
			"iat": time.Now().Unix(),
		}

		// 2. Создание нового токена с алгоритмом HS256
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

		// 3. Подписание токена секретным ключом
		tokenString, err := token.SignedString([]byte(cfg.JWT.Secret))
		if err != nil {
			FailC(c, Internal("Failed to generate JWT", err))
			return
		}

		// 4. Возврат токена клиенту
		c.JSON(http.StatusOK, gin.H{
			"token": tokenString,
		})
	}
}
