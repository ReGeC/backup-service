package config

type LocalConfig struct {
	LocalStorageEnable bool
	LocalStoragePath   string
}

func (l *LocalConfig) LoadConfig() (bool, error) {
	l.LocalStorageEnable = getEnvAsBool("LOCAL_STORAGE_ENABLE", false)

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
