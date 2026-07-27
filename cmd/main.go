package main

import (
	"log"

	"backup-service/internal/backup"
	"backup-service/internal/config"
	"backup-service/internal/storage"
)

func main() {
	cfg, err := config.NewBackupConfig()
	if err != nil {
		log.Fatal("Ошибка конфигурационного файла: ", err)
	}

	err = storage.InitDB()
	if err != nil {
		log.Fatal("Ошибка инициализации БД: ", err)
	}

	log.Println("Сервис Бекапов запущен")

	st, err := storage.NewStorage(storage.StorageType(cfg.StorageType))

	// Инициализация бэкапов
	backuppers := backup.InitBackuppers()

	// Запуск всех бэкапов
	backup.RunBackuppers(backuppers, cfg.BackupPath, cfg.StorageType)
}
