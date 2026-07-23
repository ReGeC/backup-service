package config

import (
	"log"
	"os"
	"strconv"
	"github.com/joho/godotenv"
)

func init() {
    if err := godotenv.Load(".env"); err != nil {
        log.Println(".env файл не найден, используются переменные по умолчанию")
    }
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); value != "" && exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	strValue := getEnv(key, "")
	if value, err := strconv.Atoi(strValue); err == nil {
		return value;
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	strValue := getEnv(key, "")
	if value, err := strconv.ParseBool(strValue); err == nil {
		return value;
	}
	return defaultValue
}