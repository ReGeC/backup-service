package config

import (
	"os"
	"strconv"
)

type ConfigLoader interface {
	GetEnv(key, defaultValue string) string
	GetEnvAsInt(key string, defaulValue int) int
	GetEnvAsBool(key string, defaultValue bool) bool
}

type EnvLoader struct{}

func (e *EnvLoader) GetEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func (e *EnvLoader) GetEnvAsInt(key string, defaultValue int) int {
	strValue := e.GetEnv(key, "")
	if value, err := strconv.Atoi(strValue); err == nil {
		return value
	}
	return defaultValue
}

func (e *EnvLoader) GetEnvAsBool(key string, defaultValue bool) bool {
	strValue := e.GetEnv(key, "")
	if value, err := strconv.ParseBool(strValue); err == nil {
		return value
	}
	return defaultValue
}
