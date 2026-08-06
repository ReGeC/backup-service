package config

import (
	"os"
	"strconv"
	"strings"
)

//go:generate mockery
type ConfigLoader interface {
	GetString(path []string, defaultValue string) string
	GetInt(path []string, defaultValue int) int
	GetBool(path []string, defaultValue bool) bool
}

type EnvLoader struct{}

func (e *EnvLoader) getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func (e *EnvLoader) GetString(path []string, defaultValue string) string {
	key := strings.Join(path, "_")
	return e.getEnv(key, defaultValue)
}

func (e *EnvLoader) GetInt(path []string, defaultValue int) int {
	key := strings.Join(path, "_")
	strValue := e.getEnv(key, "")
	if value, err := strconv.Atoi(strValue); err == nil {
		return value
	}
	return defaultValue
}

func (e *EnvLoader) GetBool(path []string, defaultValue bool) bool {
	key := strings.Join(path, "_")
	strValue := e.getEnv(key, "")
	if value, err := strconv.ParseBool(strValue); err == nil {
		return value
	}
	return defaultValue
}
