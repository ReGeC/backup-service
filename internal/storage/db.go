package storage

import (
	"backup-service/internal/models"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() error {
	var err error
	DB, err = gorm.Open(sqlite.Open("backup_service.db"), &gorm.Config{})
	if err != nil {
		return err
	}

	err = DB.AutoMigrate(&models.BackupLog{})
	if err != nil {
		return err
	}

	log.Println("База данных подключена")
	return nil
}

func GetDB() *gorm.DB {
	return DB
}