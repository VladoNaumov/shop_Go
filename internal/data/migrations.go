package data

import (
	"fmt"
	"os"
	"path/filepath"

	"myApp/internal/core"

	"github.com/jmoiron/sqlx"

	_ "github.com/go-sql-driver/mysql"
)

// Migrations управляет версиями БД
type Migrations struct {
	db *sqlx.DB
}

// NewMigrations создаёт мигратор
func NewMigrations(db *sqlx.DB) *Migrations {
	return &Migrations{db: db}
}

// RunMigrations выполняет все миграции
func (m *Migrations) RunMigrations() error {
	// Создаёт таблицу миграций, если не существует
	if err := m.createMigrationsTable(); err != nil {
		return err
	}

	// Находит все SQL файлы миграций
	files, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		return fmt.Errorf("ошибка поиска миграций: %w", err)
	}

	// Сортирует по номеру (001, 002...)
	files.SortByName()

	for _, file := range files {
		if err := m.runMigration(file); err != nil {
			return fmt.Errorf("ошибка миграции %s: %w", file, err)
		}
	}

	core.LogInfo("Миграции завершены успешно", map[string]interface{}{
		"files": len(files),
	})
	return nil
}

// createMigrationsTable создаёт таблицу для отслеживания миграций
func (m *Migrations) createMigrationsTable() error {
	const q = `
		CREATE TABLE IF NOT EXISTS migrations (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`

	_, err := m.db.Exec(q)
	return err
}

// runMigration выполняет одну миграцию
func (m *Migrations) runMigration(file string) error {
	name := filepath.Base(file)

	// Проверяет, применена ли уже миграция
	var exists bool
	err := m.db.Get(&exists, "SELECT COUNT(*) FROM migrations WHERE name = ?", name)
	if err != nil {
		return fmt.Errorf("ошибка проверки миграции: %w", err)
	}

	if exists {
		core.LogInfo("Миграция уже применена", map[string]interface{}{"file": name})
		return nil
	}

	// Читает SQL файл
	sqlBytes, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла: %w", err)
	}

	// Выполняет SQL (поддерживает multi-statements)
	tx, err := m.db.BeginTxx(m.db.Context(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(string(sqlBytes)); err != nil {
		return fmt.Errorf("ошибка выполнения SQL: %w", err)
	}

	// Записывает в таблицу миграций
	_, err = tx.Exec("INSERT INTO migrations (name) VALUES (?)", name)
	if err != nil {
		return fmt.Errorf("ошибка записи миграции: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка коммита: %w", err)
	}

	core.LogInfo("Миграция применена", map[string]interface{}{"file": name})
	return nil
}
