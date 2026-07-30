package main

import (
	"log"
	"context"

	"backup-service/internal/backup"
	"backup-service/internal/config"
	"backup-service/internal/storage"
	"backup-service/internal/notifier"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Println(".env файл не найден, используются переменные по умолчанию")
	}

	cfg, err := config.NewBackupConfig()
	if err != nil {
		log.Fatal("Ошибка конфигурационного файла: ", err)
	}

	err = storage.InitDB()
	if err != nil {
		log.Fatal("Ошибка инициализации БД: ", err)
	}

	log.Println("Сервис Бекапов запущен")

	// Инициализация хранилища
	st, err := storage.NewStorage(cfg.StorageType)
	if err != nil {
		log.Fatal("Ошибка создания хранилища: ", err)
	}

	//Инициализация уведомлений
	notifiers := notifier.InitNotifiers()

	// Инициализация бэкапов
	backuppers := backup.InitBackuppers()

	ctx := context.Background()

	for typ, backupper := range backuppers {
		// Создание бэкапа
		localPath, err := backup.RunBackup(typ, backupper, cfg.BackupPath, cfg.StorageType)
		if err != nil {
			msg := "Ошибка создания бэкапа " + typ + ": " + err.Error()
			notifier.SendAll(notifiers, ctx, msg)
			continue
		}

		// Сохранение в хранилище
		remotePath, err := st.Save(ctx, localPath)
		if err != nil {
			msg := "Ошибка сохранения бэкапа " + typ + " в хранилище: " + err.Error()
			notifier.SendAll(notifiers, ctx, msg)
			continue
		}

		// Уведомление об успехе
		msg := "Бэкап " + typ + " сохранен: " + remotePath
		notifier.SendAll(notifiers, ctx, msg)

		log.Printf("Бэкап %s сохранен в хранилище: %s", typ, remotePath)
	}
}
