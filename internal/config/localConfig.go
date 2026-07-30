package config

import "errors"

var ErrEmptyLocalStoragePath = errors.New("local storage path is empty")

type LocalConfig struct {
	loader ConfigLoader

	LocalStoragePath string
}

func (l *LocalConfig) LoadConfig() (bool, error) {
	// Не добаляем Enable, так как хранилище выбирается только одно
	// а не подключается как модуль
	// Оставлено конструкция как у других конфигов на будущее, если вдруг потребуется
	l.LocalStoragePath = l.loader.GetEnv("LOCAL_STORAGE_PATH", "./backups")

	if err := l.ValidateConfig(); err != nil {
		return false, err
	}

	return true, nil
}

func (l *LocalConfig) ValidateConfig() error {
	if l.LocalStoragePath == "" {
		return ErrEmptyLocalStoragePath
	}

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
