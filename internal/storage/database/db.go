package database

import (
	"backup-service/internal/models"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func InitDB() (*gorm.DB, error) {
	var err error
	DB, err := gorm.Open(sqlite.Open("backup_service.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = DB.AutoMigrate(&models.BackupLog{})
	if err != nil {
		return nil, err
	}

	log.Println("База данных подключена")
	return DB, nil
}