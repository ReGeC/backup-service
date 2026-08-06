package database

import (
	"backup-service/internal/models"
	"log/slog"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func InitDB(dbname string) (*gorm.DB, error) {
	var err error
	DB, err := gorm.Open(sqlite.Open(dbname), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = DB.AutoMigrate(&models.BackupLog{})
	if err != nil {
		return nil, err
	}

	slog.Info("База данных подключена")
	return DB, nil
}
