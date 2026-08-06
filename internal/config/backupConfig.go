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
	ErrEmptyBackupPath  = errors.New("backup path is empty")
	ErrInvalidRetention = errors.New("retention days must be greater than 0")
	ErrEmptySchedule    = errors.New("backup schedule is empty")
)

func (b *BackupConfig) LoadConfig() error {
	b.StorageType = b.loader.GetString([]string{"backup", "storage"}, "local")
	b.BackupPath = b.loader.GetString([]string{"backup", "temp_path"}, "./tmp/backups")
	b.RetentionDays = b.loader.GetInt([]string{"backup", "retention_days"}, 7)
	b.CronEnable = b.loader.GetBool([]string{"cron", "enable"}, false)
	b.BackupSchedule = b.loader.GetString([]string{"cron", "schedule"}, "0 0 3 * * *")

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

func NewBackupConfig(configPath string) (*BackupConfig, error) {
	defaultConfigPath = configPath
	loader, err := NewConfigLoader(defaultConfigPath)
	if err != nil {
		return nil, err
	}
	return NewBackupConfigWithLoader(loader)
}

func NewBackupConfigWithLoader(loader ConfigLoader) (*BackupConfig, error) {
	backupCfg := &BackupConfig{loader: loader}
	err := backupCfg.LoadConfig()
	return backupCfg, err
}
