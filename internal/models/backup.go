package models

import (
	"gorm.io/gorm"
	"time"
)

type BackupLog struct {
	ID        uint      `gorm:"primaryKey"`
	Name      string    `gorm:"index"`
	Size      int64     // размер в байтах
	Storage   string    // "local" или "s3"
	Status    string    // "success" или "failed"
	Error     string    // если была ошибка
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (BackupLog) TableName() string {
	return "backup_logs"
}
