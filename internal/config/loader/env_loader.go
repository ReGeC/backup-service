package loader

import (
	"os"
	"strconv"
	"strings"
)

type EnvLoader struct{}

func (e *EnvLoader) getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func (e *EnvLoader) GetString(path []string, defaultValue string) string {
	key := strings.ToUpper(strings.Join(path, "_"))
	return e.getEnv(key, defaultValue)
}

func (e *EnvLoader) GetInt(path []string, defaultValue int) int {
	key := strings.ToUpper(strings.Join(path, "_"))
	strValue := e.getEnv(key, "")
	if value, err := strconv.Atoi(strValue); err == nil {
		return value
	}
	return defaultValue
}

func (e *EnvLoader) GetBool(path []string, defaultValue bool) bool {
	key := strings.ToUpper(strings.Join(path, "_"))
	strValue := e.getEnv(key, "")
	if value, err := strconv.ParseBool(strValue); err == nil {
		return value
	}
	return defaultValue
}
