// internal/core/logger.go
package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type Logger struct {
	mainLogger  zerolog.Logger // ← НЕ *zerolog.Logger
	errorLogger zerolog.Logger // ← НЕ *zerolog.Logger
	mainFile    *os.File
	errorFile   *os.File
	mu          sync.Mutex
}

var (
	globalLogger *Logger
	cleanupOnce  sync.Once
)

// InitDailyLog — инициализация с ротацией по дням
func InitDailyLog() {
	// Закрываем старые файлы
	if globalLogger != nil {
		globalLogger.mu.Lock()
		_ = globalLogger.mainFile.Close()
		_ = globalLogger.errorFile.Close()
		globalLogger.mu.Unlock()
		globalLogger = nil
	}

	// Создаём директорию
	if err := os.MkdirAll("logs", 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка создания logs: %v\n", err)
		os.Exit(1)
	}

	// Имена файлов по дате
	dateStr := time.Now().Format("02-01-2006")
	mainPath := filepath.Join("logs", dateStr+".log")
	errorPath := filepath.Join("logs", "errors-"+dateStr+".log")

	// Открываем файлы
	mainFile, err := os.OpenFile(mainPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка открытия main.log: %v\n", err)
		os.Exit(1)
	}

	errorFile, err := os.OpenFile(errorPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		_ = mainFile.Close()
		fmt.Fprintf(os.Stderr, "Ошибка открытия error.log: %v\n", err)
		os.Exit(1)
	}

	// ← ИСПРАВЛЕНО: MultiWriter + консоль
	mainWriter := zerolog.MultiLevelWriter(os.Stdout, mainFile)
	errorWriter := zerolog.MultiLevelWriter(os.Stderr, errorFile)

	// ← ИСПРАВЛЕНО: zerolog.Logger, НЕ *zerolog.Logger
	mainLogger := zerolog.New(mainWriter).With().Timestamp().Logger()
	errorLogger := zerolog.New(errorWriter).With().Timestamp().Logger()

	globalLogger = &Logger{
		mainLogger:  mainLogger,  // ← Прямая передача значения
		errorLogger: errorLogger, // ← Прямая передача значения
		mainFile:    mainFile,
		errorFile:   errorFile,
	}

	// Очистка старых логов (один раз)
	cleanupOnce.Do(func() {
		go cleanupOldLogs("logs", 7)
	})
}

// LogInfo — с fallback в stdout
func LogInfo(msg string, fields map[string]interface{}) {
	if globalLogger == nil {
		// ← ИСПРАВЛЕНО: zerolog.New возвращает *zerolog.Logger → вызываем через переменную
		l := zerolog.New(os.Stdout).With().Timestamp().Logger()
		event := l.Info()
		for k, v := range fields {
			event = event.Interface(k, v)
		}
		event.Msg(msg)
		return
	}

	globalLogger.mu.Lock()
	defer globalLogger.mu.Unlock()

	event := globalLogger.mainLogger.Info() // ← mainLogger — zerolog.Logger → OK
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

// LogError — с fallback в stderr
func LogError(msg string, fields map[string]interface{}) {
	if globalLogger == nil {
		// ← ИСПРАВЛЕНО: аналогично
		l := zerolog.New(os.Stderr).With().Timestamp().Logger()
		event := l.Error()
		for k, v := range fields {
			event = event.Interface(k, v)
		}
		event.Msg(msg)
		return
	}

	globalLogger.mu.Lock()
	defer globalLogger.mu.Unlock()

	event := globalLogger.errorLogger.Error() // ← errorLogger — zerolog.Logger → OK
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

// cleanupOldLogs — удаление логов старше N дней
func cleanupOldLogs(dir string, days int) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		info, err := file.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			path := filepath.Join(dir, file.Name())
			_ = os.Remove(path)
		}
	}
}

// Close — закрытие файлов
func Close() {
	if globalLogger == nil {
		return
	}
	globalLogger.mu.Lock()
	defer globalLogger.mu.Unlock()

	_ = globalLogger.mainFile.Close()
	_ = globalLogger.errorFile.Close()
	globalLogger = nil
}
