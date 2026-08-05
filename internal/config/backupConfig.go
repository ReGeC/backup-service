package config

import (
	"errors"
)

type BackupConfig struct {
	loader ConfigLoader

	StorageType   string
	BackupPath    string
	RetentionDays int

	CronEnable     bool
	BackupSchedule string
}

var (
	ErrEmptyBackupPath = errors.New("backup path is empty")
	ErrInvalidRetention = errors.New("retention days must be greater than 0")
	ErrEmptySchedule = errors.New("backup schedule is empty")
)



func (b *BackupConfig) LoadConfig() error {
	b.StorageType = b.loader.GetEnv("BACKUP_STORAGE", "local")
	b.BackupPath = b.loader.GetEnv("BACKUP_TEMP_PATH", "./tmp/backups")
	b.RetentionDays = b.loader.GetEnvAsInt("BACKUP_RETENTION_DAYS", 7)
	b.CronEnable = b.loader.GetEnvAsBool("CRON_ENABLE", false)
	b.BackupSchedule = b.loader.GetEnv("CRON_SCHEDULE", "0 0 3 * * *")

	return b.ValidateConfig()
}

func (b *BackupConfig) ValidateConfig() error {
	if b.BackupPath == "" {
		return ErrEmptyBackupPath
	}
	if b.RetentionDays <= 0 {
		return ErrInvalidRetention
	}
	if b.CronEnable && b.BackupSchedule == "" {
		return ErrEmptySchedule
	}

	return nil
}

func NewBackupConfig() (*BackupConfig, error) {
	return NewBackupConfigWithLoader(&EnvLoader{})
}

func NewBackupConfigWithLoader(loader ConfigLoader) (*BackupConfig, error) {
	backupCfg := &BackupConfig{loader: loader}
	err := backupCfg.LoadConfig()
	return backupCfg, err
}
