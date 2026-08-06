package backup

import (
	"backup-service/internal/config"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
)

const SQLite = "sqlite"

var ErrSQLiteDisabled = errors.Join(ErrDisabled, errors.New(SQLite))

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
		return nil, ErrSQLiteDisabled
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

	fullPath = buildBackupPath(SQLite, outputDir)

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

func (s *SQLiteBackup) RestoreBackup(ctx context.Context, backupPath string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// Проверяем наличие бэкапа
	if _, err := os.Stat(backupPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("файл бэкапа не найден: %s", backupPath)
		}
		return "", fmt.Errorf("проверка файла бэкапа: %w", err)
	}

	// Формируем имя восстановленной БД
	restoredPath := buildRestoredPath(s.DBPath) + ".db"

	srcFile, err := os.Open(backupPath)
	if err != nil {
		return "", fmt.Errorf("ошибка открытия бэкапа: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(restoredPath)
	if err != nil {
		return "", fmt.Errorf("ошибка создания восстановленной БД: %w", err)
	}

	defer func() {
		if closeErr := dstFile.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("ошибка закрытия файла: %w", closeErr)
		}

		if err != nil {
			_ = os.Remove(restoredPath)
		}
	}()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return "", fmt.Errorf("ошибка восстановления БД: %w", err)
	}

	return restoredPath, nil
}

func (s *SQLiteBackup) GetBackupType() string {
	return SQLite
}
