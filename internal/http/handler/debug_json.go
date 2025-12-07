package handler

import (
	"fmt"
	"net/http"
	"time"

	"myApp/internal/core"
	"myApp/internal/storage"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// Debug — возвращает расширенную отладочную информацию в JSON (latency, session, nonce, DB ping)
func Debug(c *gin.Context) {
	startTime := time.Now()

	info := map[string]interface{}{
		"request": map[string]interface{}{
			"method":  c.Request.Method,
			"url":     c.Request.URL.String(),
			"headers": c.Request.Header,
			"remote":  c.Request.RemoteAddr,
		},
		"processing": map[string]interface{}{
			"start_time": startTime.Format(time.RFC3339),
			"request_id": c.GetHeader("X-Request-Id"), // Работает!
			"latency_ms": time.Since(startTime).Milliseconds(),
		},
		"session": map[string]interface{}{
			"exists": false,
		},
		"context": map[string]interface{}{
			"nonce": c.GetString(string(core.CtxNonce)), // Если CtxNonce type, cast
		},
		"health": map[string]interface{}{
			"db_ping": "pending",
		},
		"response": map[string]interface{}{
			"content_type": "application/json",
			"status":       http.StatusOK,
			"note":         "OK! (расширенная отладка)",
		},
	}

	// Сессия
	if session := sessions.Default(c); session != nil {
		info["session"].(map[string]interface{})["exists"] = true
	}

	// DB ping (простой error, как раньше)
	if dbIface, ok := c.Get(storage.CtxDBKey{}); ok {
		if db, ok := dbIface.(*sqlx.DB); ok && db != nil {
			var ping int
			var err = db.Get(&ping, "SELECT 1")
			healthInfo := info["health"].(map[string]interface{})
			if err != nil {
				healthInfo["db_ping"] = fmt.Sprintf("error: %v", err)
			} else {
				healthInfo["db_ping"] = fmt.Sprintf("ok (ping: %d)", ping)
			}
			info["context"].(map[string]interface{})["db_connected"] = true
		} else {
			info["context"].(map[string]interface{})["db_connected"] = false
			info["health"].(map[string]interface{})["db_ping"] = "error: no DB in context"
		}
	} else {
		info["context"].(map[string]interface{})["db_connected"] = false
		info["health"].(map[string]interface{})["db_ping"] = "error: no DB in context"
	}

	core.JSON(c, http.StatusOK, info)
}
