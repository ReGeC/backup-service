package database

import (
	"backup-service/internal/models"

	"gorm.io/gorm"
)

type GormBackupLogRepository struct {
	db *gorm.DB
}

func NewGormBackupLogRepository(db *gorm.DB) *GormBackupLogRepository {
	return &GormBackupLogRepository{
		db: db,
	}
}

func (r *GormBackupLogRepository) CreateLog(log *models.BackupLog) error {
	return r.db.Create(log).Error
}
