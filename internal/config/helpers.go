package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type ConfigLoader interface {
	GetEnv(key, defaultValue string) string
	GetEnvAsInt(key string, defaulValue int) int
	GetEnvAsBool(key string, defaultValue bool) bool
}

type EnvLoader struct{}

func init() {
	if err := godotenv.Load(".env"); err != nil {
		log.Println(".env файл не найден, используются переменные по умолчанию")
	}
}

func (e *EnvLoader) GetEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); value != "" && exists {
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
