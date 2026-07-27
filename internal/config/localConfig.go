package config

type LocalConfig struct {
	loader ConfigLoader

	LocalStoragePath string
}

func (l *LocalConfig) LoadConfig() (bool, error) {
	// Не добаляем Enable, так как хранилище выбирается только одно
	// а не подключается как модуль
	l.LocalStoragePath = l.loader.GetEnv("LOCAL_STORAGE_PATH", "./backups")

	err := l.ValidateConfig()

	return true, err
}

// TODO Валидация конфига
func (l *LocalConfig) ValidateConfig() error {
	return nil
}

func NewLocalConfig() (*LocalConfig, bool, error) {
	return NewLocalConfigWithLoader(&EnvLoader{})
}

func NewLocalConfigWithLoader(loader ConfigLoader) (*LocalConfig, bool, error) {
	localConfig := &LocalConfig{loader: loader}
	enabled, err := localConfig.LoadConfig()
	return localConfig, enabled, err
}
