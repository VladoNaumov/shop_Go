package handler

import (
	"fmt"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"time"

	"myApp/internal/core"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// Переменные, которые могут быть установлены через ldflags при сборке
var (
	AppVersion   = "dev"
	GoVersion    = runtime.Version()
	appStartTime = time.Now()
)

// Константы для ключей Gin Context (должны совпадать с main.go)
const (
	ContextNonceKey = "csp_nonce"
	ContextDBKey    = "db_connection"
	SchemaVersion   = "1.0"
)

// tryExtractSessionValues пытается "best-effort" получить полное содержимое сессии.
// Возвращает map[string]interface{} или nil, если извлечение не удалось.
func tryExtractSessionValues(sess sessions.Session) map[string]interface{} {
	if sess == nil {
		return nil
	}

	sv := reflect.ValueOf(sess)
	defer func() { _ = recover() }() // защищаемся от паники при вызове рефлексии

	methodCandidates := []string{"All", "GetAll", "Values"}
	for _, mName := range methodCandidates {
		m := sv.MethodByName(mName)
		if m.IsValid() && m.Type().NumIn() == 0 && m.Type().NumOut() == 1 {
			out := m.Call(nil)
			if len(out) == 1 && out[0].Kind() == reflect.Map {
				return mapFromReflectMap(out[0])
			}
		}
	}

	rv := sv
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.IsValid() {
		fieldNames := []string{"Values", "values"}
		for _, fname := range fieldNames {
			f := rv.FieldByName(fname)
			if f.IsValid() && f.Kind() == reflect.Map {
				return mapFromReflectMap(f)
			}
		}
		otherNames := []string{"session", "Session", "data", "Data", "store"}
		for _, fname := range otherNames {
			f := rv.FieldByName(fname)
			if f.IsValid() {
				if f.Kind() == reflect.Struct {
					sub := f
					if sub.Kind() == reflect.Ptr {
						sub = sub.Elem()
					}
					for _, sf := range []string{"Values", "values", "Data", "data"} {
						sfField := sub.FieldByName(sf)
						if sfField.IsValid() && sfField.Kind() == reflect.Map {
							return mapFromReflectMap(sfField)
						}
					}
				}
				if f.Kind() == reflect.Map {
					return mapFromReflectMap(f)
				}
			}
		}
	}

	return nil
}

// mapFromReflectMap преобразует reflect.Value (map) в map[string]interface{} с конвертацией ключей в строки.
func mapFromReflectMap(m reflect.Value) map[string]interface{} {
	if !m.IsValid() || m.Kind() != reflect.Map {
		return nil
	}
	out := make(map[string]interface{}, m.Len())
	for _, k := range m.MapKeys() {
		ks := fmt.Sprint(k.Interface())
		val := m.MapIndex(k).Interface()
		out[ks] = val
	}
	return out
}

// Debug — handler, возвращающий расширенную отладочную информацию в JSON.
func Debug(c *gin.Context) {
	startTime := time.Now()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	rawCookies := map[string]string{}
	for _, ck := range c.Request.Cookies() {
		rawCookies[ck.Name] = ck.Value
	}

	var rawSessionCookie string
	if ck, err := c.Request.Cookie("mysession"); err == nil {
		rawSessionCookie = ck.Value
	}

	uptime := time.Since(appStartTime).Seconds()

	info := map[string]interface{}{
		"schema_version": SchemaVersion,
		"app_info": map[string]interface{}{
			"version":        AppVersion,
			"go_version":     GoVersion,
			"os_arch":        fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			"go_routines":    runtime.NumGoroutine(),
			"uptime_seconds": uptime,
		},
		"request": map[string]interface{}{
			"method":    c.Request.Method,
			"url":       c.Request.URL.String(),
			"headers":   c.Request.Header,
			"remote":    c.Request.RemoteAddr,
			"remote_ip": c.ClientIP(),
			"full_path": c.FullPath(),
			"cookies":   rawCookies,
		},
		"processing": map[string]interface{}{
			"start_time": startTime.Format(time.RFC3339Nano),
			"latency_ms": 0,
		},
		"context": map[string]interface{}{
			"nonce":        c.GetString(ContextNonceKey),
			"db_connected": false,
		},
		"health": map[string]interface{}{
			"db_ping": "pending",
		},
		"resources": map[string]interface{}{
			"alloc_mb":    float64(memStats.Alloc) / 1024.0 / 1024.0,
			"sys_mb":      float64(memStats.Sys) / 1024.0 / 1024.0,
			"gc_count":    memStats.NumGC,
			"gc_pause_ms": float64(memStats.PauseTotalNs) / float64(time.Millisecond),
		},
		"response": map[string]interface{}{
			"content_type": "application/json",
			"status":       http.StatusOK,
			"note":         "OK! (Расширенная отладка)",
		},
	}

	sessionInfo := map[string]interface{}{
		"exists":             false,
		"session_cookie_raw": rawSessionCookie,
		"authenticated":      false,
		"values":             nil,
	}

	if sess := sessions.Default(c); sess != nil {
		sessionInfo["exists"] = true
		if vals := tryExtractSessionValues(sess); vals != nil {
			sessionInfo["values"] = vals
			if _, ok := vals["user_id"]; ok {
				sessionInfo["authenticated"] = true
			}
		} else {
			known := []string{"user_id", "user", "email", "authenticated", "is_admin", "csrf"}
			fallback := map[string]interface{}{}
			for _, k := range known {
				if v := sess.Get(k); v != nil {
					fallback[k] = v
					if k == "user_id" {
						sessionInfo["authenticated"] = true
					}
				}
			}
			sessionInfo["values"] = fallback
		}
	}

	info["session"] = sessionInfo

	if dbIface, ok := c.Get(ContextDBKey); ok {
		if db, ok := dbIface.(*sqlx.DB); ok && db != nil {
			startPing := time.Now()
			err := db.Ping()
			health := info["health"].(map[string]interface{})
			if err != nil {
				health["db_ping"] = fmt.Sprintf("error: %v", err)
			} else {
				stats := db.Stats()
				health["db_pool_open"] = stats.OpenConnections
				health["db_pool_in_use"] = stats.InUse
				health["db_idle"] = stats.Idle
				health["db_ping"] = fmt.Sprintf("ok (latency: %dms)", time.Since(startPing).Milliseconds())
			}
			info["context"].(map[string]interface{})["db_connected"] = true
		} else {
			info["health"].(map[string]interface{})["db_ping"] = "error: DB interface conversion failed"
		}
	} else {
		info["health"].(map[string]interface{})["db_ping"] = "error: no DB found in Gin context (key: db_connection)"
	}

	info["processing"].(map[string]interface{})["latency_ms"] = time.Since(startTime).Milliseconds()

	hostname, _ := os.Hostname()
	info["server"] = map[string]interface{}{
		"hostname": hostname,
		"now":      time.Now().Format(time.RFC3339Nano),
		"pid":      os.Getpid(),
	}

	core.JSON(c, http.StatusOK, info)
}
