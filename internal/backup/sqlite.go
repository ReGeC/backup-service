package backup

import (
	"backup-service/internal/config"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const SQLite = "sqlite"

func init() {
	// Автоматическая регистрация при импорте
	cfg, enabled, err := config.NewSQLiteConfig()
	if enabled {
		Register(SQLite, func() (Backupper, error) {
			if err != nil {
				return nil, fmt.Errorf("Неверная конфигурация для SQLite: %w", err)
			}
			return NewSQLiteBackupFromConfig(cfg), nil
		})
	}
}

type SQLiteBackup struct {
	DBPath string
}

func NewSQLiteBackup(dbPath string) *SQLiteBackup {
	return &SQLiteBackup{
		DBPath: dbPath,
	}
}

func NewSQLiteBackupFromConfig(cfg *config.SQLiteConfig) *SQLiteBackup {
	return &SQLiteBackup{
        DBPath: cfg.SQLitePath,
    }
}

func (s *SQLiteBackup) CreateBackup(ctx context.Context, outputDir string) (fullPath string, err error) {
	timestamp := time.Now().Format("2006-01-02_15-04")
	filename := fmt.Sprintf("sqlite_%s.db.bak", timestamp)
	fullPath = filepath.Join(outputDir, filename)

	// Проекра на существование БД
	if _, err := os.Stat(s.DBPath); os.IsNotExist(err) {
		return "", fmt.Errorf("Файл БД не найден: %s", s.DBPath)
	}

	// Копируем файл
	srcFile, err := os.Open(s.DBPath)
	if err != nil {
		return "", fmt.Errorf("Ошибка открытия исходной БД: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(fullPath)
	if err != nil {
	    return "", fmt.Errorf("ошибка создания бэкапа: %w", err)
	}
	// Обработка ошибки записи в файл
	defer func() {
	    if closeErr := dstFile.Close(); err == nil && closeErr != nil {
	        err = fmt.Errorf("ошибка закрытия файла: %w", closeErr)
	    }

	    if err != nil {
	        _ = os.Remove(fullPath)
	    }
	}()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return "", fmt.Errorf("Ошибка копирования: %w", err)
	}

	return fullPath, nil
}

func (s* SQLiteBackup) GetBackupType() string {
	return SQLite
}