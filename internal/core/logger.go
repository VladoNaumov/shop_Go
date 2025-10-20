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
	mainLogger  *zerolog.Logger
	errorLogger *zerolog.Logger
	mainFile    *os.File
	errorFile   *os.File
	mu          sync.Mutex
}

var (
	globalLogger *Logger
	cleanupOnce  sync.Once
)

// Инициализация ежедневного Log журнала
func InitDailyLog() {
	// Закрываем предыдущие файлы, если есть
	if globalLogger != nil {
		globalLogger.mu.Lock()
		_ = globalLogger.mainFile.Close()
		_ = globalLogger.errorFile.Close()
		globalLogger.mu.Unlock()
		globalLogger = nil
	}

	// Создаём директорию logs
	if err := os.MkdirAll("logs", 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка создания директории logs: %v\n", err)
		os.Exit(1)
	}

	// Формируем имена файлов на основе текущей даты
	dateStr := time.Now().Format("02-01-2006")
	mainPath := filepath.Join("logs", dateStr+".log")
	errorPath := filepath.Join("logs", "errors-"+dateStr+".log")

	// Открываем файлы
	mainFile, err := os.OpenFile(mainPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка открытия основного лог-файла: %v\n", err)
		os.Exit(1)
	}

	errorFile, err := os.OpenFile(errorPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка открытия файла ошибок: %v\n", err)
		_ = mainFile.Close()
		os.Exit(1)
	}

	// Настраиваем Zerolog для основного лога
	mainLogger := zerolog.New(mainFile).With().Timestamp().Logger()

	// Настраиваем Zerolog для логов ошибок
	errorLogger := zerolog.New(errorFile).With().Timestamp().Logger()

	globalLogger = &Logger{
		mainLogger:  &mainLogger,
		errorLogger: &errorLogger,
		mainFile:    mainFile,
		errorFile:   errorFile,
	}

	// Запускаем очистку старых логов один раз
	cleanupOnce.Do(func() { go cleanupOldLogs("logs", 7) })
}

func LogInfo(msg string, fields map[string]interface{}) {
	if globalLogger == nil {
		return // Игнорируем, если логгер закрыт
	}
	globalLogger.mu.Lock()
	defer globalLogger.mu.Unlock()

	event := globalLogger.mainLogger.Info()
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

func LogError(msg string, fields map[string]interface{}) {
	if globalLogger == nil {
		return // Игнорируем, если логгер закрыт
	}
	globalLogger.mu.Lock()
	defer globalLogger.mu.Unlock()

	event := globalLogger.errorLogger.Error()
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

func cleanupOldLogs(dir string, days int) {
	files, err := os.ReadDir(dir)
	if err != nil {
		if globalLogger != nil {
			globalLogger.errorLogger.Error().Msgf("Не удалось прочитать %s: %v", dir, err)
		}
		return
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		info, err := file.Info()
		if err != nil {
			if globalLogger != nil {
				globalLogger.errorLogger.Error().Msgf("Инфо о %s: %v", file.Name(), err)
			}
			continue
		}
		if info.ModTime().Before(cutoff) {
			path := filepath.Join(dir, file.Name())
			if err := os.Remove(path); err != nil {
				if globalLogger != nil {
					globalLogger.errorLogger.Error().Msgf("Удаление %s: %v", path, err)
				}
			}
		}
	}
}

func Close() {
	if globalLogger != nil {
		globalLogger.mu.Lock()
		defer globalLogger.mu.Unlock()

		// Логируем ошибки закрытия в stderr
		consoleLogger := zerolog.New(os.Stderr).With().Timestamp().Logger()
		if err := globalLogger.mainFile.Close(); err != nil {
			consoleLogger.Error().Msgf("Закрытие mainFile: %v", err)
		}
		if err := globalLogger.errorFile.Close(); err != nil {
			consoleLogger.Error().Msgf("Закрытие errorFile: %v", err)
		}
		globalLogger = nil
	}
}
