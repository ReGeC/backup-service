package config

type LocalConfig struct {
	LocalStoragePath   string
}

func (l *LocalConfig) LoadConfig() (bool, error) {
	// Не добаляем Enable, так как хранилище выбирается только одно
	// а не подключается как модуль
	l.LocalStoragePath = getEnv("LOCAL_STORAGE_PATH", "./backups")

	err := l.ValidateConfig()

	return true, err
}

// TODO Валидация конфига
func (l *LocalConfig) ValidateConfig() error {
	return nil
}

func NewLocalConfig() (*LocalConfig, bool, error) {
	localConfig := &LocalConfig{}
	enabled, err := localConfig.LoadConfig()
	return localConfig, enabled, err
}
