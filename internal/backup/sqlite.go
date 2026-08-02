package backup

import (
	"backup-service/internal/config"
	"context"
	"fmt"
	"io"
	"os"
)

const SQLite = "sqlite"

func init() {
	// Автоматическая регистрация при импорте
	Register(SQLite, newSQLiteBackupper)
}

func newSQLiteBackupper() (Backupper, error) {
    cfg, enabled, err := config.NewSQLiteConfig()
    if err != nil {
        return nil, err
    }
	
	if !enabled {
        return nil, ErrDisabled
    }

    return NewSQLiteBackup(cfg.SQLitePath), nil
}

type SQLiteBackup struct {
	DBPath string
}

func NewSQLiteBackup(dbPath string) *SQLiteBackup {
	return &SQLiteBackup{
		DBPath: dbPath,
	}
}

func (s *SQLiteBackup) CreateBackup(ctx context.Context, outputDir string) (fullPath string, err error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	
	fullPath = buildBackupPath(outputDir)

	// Проверка на существование БД
	if _, err := os.Stat(s.DBPath); err != nil {
        if os.IsNotExist(err) {
            return "", fmt.Errorf("файл БД не найден: %s", s.DBPath)
        }
        return "", fmt.Errorf("проверка файла БД: %w", err)
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
	        _ = os.Remove(dstFile.Name())
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