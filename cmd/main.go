package main

import (
	"log"
	"time"
	"os"

	"backup-service/internal/backup"
	"backup-service/internal/config"
	"backup-service/internal/storage"
	"backup-service/internal/models"
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

	// Инициализация бэкапов
	backuppers := map[string]backup.Backupper{}
	if cfg.PGEnable {
		pgBackup, err := backup.NewBackupper(backup.Postgres)
		if err != nil {
			log.Fatal("Ошибка инициализации Postgres: ", err)
		}
		backuppers["postgres"] = pgBackup
	}
	if cfg.SQLiteEnable {
		sqliteBackup, err := backup.NewBackupper(backup.SQLite)
		if err != nil {
			log.Fatal("Ошибка инициализации SQLite: ", err)
		}
		backuppers["sqlite"] = sqliteBackup
	}

	// Проверка, что хотя бы 1 бэкап включен
	if len(backuppers) == 0 {
        log.Fatal("❌ Нет включенных БД для бэкапа")
    }

	

	// Запуск всех бэкапов
	for key, backupper := range backuppers {
	    log.Printf("Создание бэкапа %s\n", key)
	    startTime := time.Now()

	    filePath, err := backupper.Create(cfg.BackupPath)
	    if err != nil {
	        log.Printf("❌ Ошибка бэкапа %s: %v", key, err)
		
	        // Логируем ошибку в БД
	        logEntry := &models.BackupLog{
	            Name:    key + "_backup",
	            Size:    0,
	            Storage: cfg.StorageType,
	            Status:  "failed",
	            Error:   err.Error(),
	        }
	        storage.GetDB().Create(logEntry)
		
	        continue // переходим к следующему бэкапу
	    }

	    // Получаем размер файла
	    fileInfo, err := os.Stat(filePath)
	    if err != nil {
	        log.Printf("⚠️ Не удалось получить размер файла %s: %v", filePath, err)
		
	        // Логируем с размером 0, но статус success (бэкап-то создан)
	        logEntry := &models.BackupLog{
	            Name:    filePath,
	            Size:    0,
	            Storage: cfg.StorageType,
	            Status:  "success",
	            Error:   "не удалось получить размер файла: " + err.Error(),
	        }
	        storage.GetDB().Create(logEntry)
		
	        elapsed := time.Since(startTime)
	        log.Printf("✅ Бэкап %s создан: %s (размер: неизвестен, время: %v)",
	            key, filePath, elapsed)
	        continue
	    }

	    // Сохраняем в лог успешный бэкап с размером
	    logEntry := &models.BackupLog{
	        Name:    filePath,
	        Size:    fileInfo.Size(),
	        Storage: cfg.StorageType,
	        Status:  "success",
	    }

	    if result := storage.GetDB().Create(logEntry); result.Error != nil {
	        log.Printf("⚠️ Ошибка сохранения лога для %s: %v", key, result.Error)
	        // Не прерываем выполнение, бэкап уже создан
	    }

	    elapsed := time.Since(startTime)
	    log.Printf("✅ Бэкап %s создан: %s (размер: %.2f MB, время: %v)",
	        key, filePath, float64(fileInfo.Size())/1024/1024, elapsed)
	}
}
