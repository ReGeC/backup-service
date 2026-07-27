package config

type BackupConfig struct {
	loader ConfigLoader

	StorageType   string
	BackupPath    string
	RetentionDays int

	CronEnable     bool
	BackupSchedule string
}

func (b *BackupConfig) LoadConfig() error {
	b.StorageType = b.loader.GetEnv("BACKUP_STORAGE", "local")
	b.BackupPath = b.loader.GetEnv("BACKUP_TEMP_PATH", "./tmp/backups")
	b.RetentionDays = b.loader.GetEnvAsInt("BACKUP_RETENTION_DAYS", 7)
	b.CronEnable = b.loader.GetEnvAsBool("CRON_ENABLE", false)
	b.BackupSchedule = b.loader.GetEnv("BACKUP_SCHEDULE", "0 3 * * *")

	err := b.ValidateConfig()

	return err
}

// TODO Валидация конфига
func (b *BackupConfig) ValidateConfig() error {
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
