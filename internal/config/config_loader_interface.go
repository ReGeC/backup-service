package config

import (
	"backup-service/internal/config/loader"
	"fmt"
)

var defaultConfigPath string

//go:generate mockery
type ConfigLoader interface {
	GetString(path []string, defaultValue string) string
	GetInt(path []string, defaultValue int) int
	GetBool(path []string, defaultValue bool) bool
}

func NewConfigLoader(configPath string) (ConfigLoader, error) {
	if configPath == "" {
		return &loader.EnvLoader{}, nil
	}

	viperLoader, err := loader.NewViperLoader(configPath)
	if err != nil {
		return nil, fmt.Errorf("create viper loader: %w", err)
	}

	return viperLoader, nil
}


