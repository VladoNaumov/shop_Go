package core

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// InitDailyLog — инициализирует лог-файлы формата logs/DD-MM-YYYY.log
// и logs/errors-DD-MM-YYYY.log.
// Старые файлы (старше 7 дней) автоматически удаляются.
func InitDailyLog() {
	// Создаём директорию logs, если её нет
	_ = os.MkdirAll("logs", 0755)

	// Форматы имен файлов: день-месяц-год
	dateStr := time.Now().Format("02-01-2006")
	mainPath := filepath.Join("logs", dateStr+".log")
	errPath := filepath.Join("logs", "errors-"+dateStr+".log")

	// --- Открываем основной лог ---
	mainFile, err := os.OpenFile(mainPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Не удалось открыть основной лог-файл: %v", err)
		return
	}

	// --- Открываем error.log ---
	errorFile, err := os.OpenFile(errPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Не удалось открыть error лог-файл: %v", err)
		return
	}

	// --- Создаём writer, который дублирует вывод ---
	logWriter := io.MultiWriter(mainFile, os.Stdout, newErrorSplitter(errorFile))

	log.SetOutput(logWriter)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("=== Новый лог-файл ===")

	// Асинхронная очистка старых логов
	go cleanupOldLogs("logs", 7)
}

// newErrorSplitter — обёртка, которая дублирует только ERROR-записи в error.log
func newErrorSplitter(errorFile *os.File) io.Writer {
	return writerFunc(func(p []byte) (n int, err error) {
		line := string(p)
		if strings.Contains(strings.ToUpper(line), "ERROR") {
			// Пишем в error.log
			_, _ = errorFile.Write(p)
		}
		return os.Stdout.Write(p) // возвращаем стандартное поведение
	})
}

type writerFunc func(p []byte) (n int, err error)

func (f writerFunc) Write(p []byte) (n int, err error) {
	return f(p)
}

// cleanupOldLogs удаляет лог-файлы старше указанного количества дней.
func cleanupOldLogs(dir string, maxAgeDays int) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -maxAgeDays)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		info, err := file.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, file.Name()))
		}
	}
}
