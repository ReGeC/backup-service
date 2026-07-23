package backup

import (
	"backup-service/internal/config"
	"fmt"
)

type Backupper interface {
	Create(outputDir string) (string, error)
}

type BackupperConfig interface {
	LoadConfig() (error)
}

type BackupType string
const(
	Postgres BackupType = "postgres"
	SQLite   BackupType = "sqlite"
)

func NewBackupper(backupType BackupType) (Backupper, error) {
	switch backupType {
	case Postgres:
		cfg, err := config.NewPostgresConfig()
		if err != nil {
			return nil, fmt.Errorf("Неверная конфигурация для PostgreSQL: %w", err)
		}
		return NewPostgresBackupFromConfig(cfg), nil
	case SQLite:
		cfg, err := config.NewSQLiteConfig()
		if err != nil {
			return nil, fmt.Errorf("Неверная конфигурация для SQLite: %w", err)
		}
		return NewSQLiteBackupFromConfig(cfg), nil
	default:
		return nil, fmt.Errorf("Непподерживаемый тип бэкапа: %s", backupType)
	}
}
