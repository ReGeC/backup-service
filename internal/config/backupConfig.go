package config

type BackupConfig struct {
	StorageType string
	BackupPath string
	RetentionDays int
	BackupSchedule string

	PGEnable bool
	SQLiteEnable bool
}

func (b *BackupConfig) LoadConfig() error {
	b.StorageType = getEnv("BACKUP_STORAGE", "local")
	b.BackupPath = getEnv("BACKUP_PATH", "./backups")
	b.RetentionDays = getEnvAsInt("BACKUP_RETENTION_DAYS", 7)
	b.BackupSchedule = getEnv("BACKUP_SCHEDULE", "0 3 * * *")
	b.PGEnable = getEnvAsBool("PG_ENABLE", false)
	b.SQLiteEnable = getEnvAsBool("SQLITE_ENABLE", false)

	err := b.ValidateConfig()

	return err
}

// TODO Валидация конфига
func (b *BackupConfig) ValidateConfig() error {
	return nil;
}

func NewBackupConfig() (*BackupConfig, error) {
	backupCfg := &BackupConfig{}
	err := backupCfg.LoadConfig()
	return backupCfg, err
}