package main

import (
	"context"
	"flag"
	"log"

	"backup-service/internal/backup"
	"backup-service/internal/config"
	"backup-service/internal/storage"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Println(".env файл не найден, используются переменные окружения")
	}

	backupName := flag.String(
		"backup",
		"",
		"имя файла бэкапа для восстановления",
	)

	backupType := flag.String(
		"type",
		"",
		"тип бэкаппера (sqlite, postgres и т.д.)",
	)

	flag.Parse()

	if *backupName == "" {
		log.Fatal("не указан файл бэкапа: -backup")
	}

	if *backupType == "" {
		log.Fatal("не указан тип бэкаппера: -type")
	}

	cfg, err := config.NewBackupConfig()
	if err != nil {
		log.Fatal("Ошибка конфигурации: ", err)
	}

	ctx := context.Background()

	// Инициализация storage
	st, err := storage.NewStorage(cfg.StorageType)
	if err != nil {
		log.Fatal("Ошибка создания хранилища: ", err)
	}

	// Инициализация backupper-ов
	backuppers := backup.InitBackuppers()

	backupper, ok := backuppers[*backupType]
	if !ok {
		log.Fatalf(
			"не найден бэкаппер типа: %s",
			*backupType,
		)
	}

	// Получаем файл из storage
	localBackupPath, err := st.Download(
		ctx,
		*backupName,
	)

	if err != nil {
		log.Fatal(
			"Ошибка получения бэкапа из хранилища: ",
			err,
		)
	}

	// Восстанавливаем
	restoredPath, err := backupper.RestoreBackup(
		ctx,
		localBackupPath,
	)

	if err != nil {
		log.Fatal(
			"Ошибка восстановления: ",
			err,
		)
	}

	log.Println("БД успешно восстановлена:")
	log.Println(restoredPath)
}
