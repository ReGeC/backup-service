// cmd/backup-service/main.go (после рефакторинга)
package main

import (
	"log/slog"
	"os"

	"backup-service/internal/app"

	"github.com/joho/godotenv"
)

func main() {
	// Устанавливаем глобальный логгер
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	if err := godotenv.Load(".env"); err != nil {
		slog.Warn(".env файл не найден, используются переменные по умолчанию")
	}

	// Создаем приложение
	appInstance, err := app.New()
	if err != nil {
		slog.Error("ошибка инициализации", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := appInstance.Close(); err != nil {
			slog.Error("ошибка закрытия ресурсов", "error", err)
		}
	}()

	// Запуск
	if err := appInstance.Start(); err != nil {
		slog.Error("ошибка запуска", "error", err)
		os.Exit(1)
	}
}
