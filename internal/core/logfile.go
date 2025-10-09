package core

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

// InitDailyLog — инициализирует лог-файл формата logs/DD-MM-YYYY.log.
// Если директории "logs" нет — создаёт её.
// Старые логи (старше 7 дней) автоматически удаляются.
func InitDailyLog() {
	// Создаём директорию logs, если её нет
	_ = os.MkdirAll("logs", 0755)

	// Формат имени: день-месяц-год
	filename := time.Now().Format("02-01-2006") + ".log"
	path := filepath.Join("logs", filename)

	// Открываем файл (создаём, если нет)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Не удалось открыть лог-файл: %v", err)
		return
	}

	// Перенаправляем стандартный лог в файл
	log.SetOutput(f)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("=== Новый лог-файл ===")

	// Запускаем очистку старых логов (асинхронно)
	go cleanupOldLogs("logs", 7)
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
