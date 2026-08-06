package database_test

import (
	"backup-service/internal/models"
	"backup-service/internal/storage/database"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGormBackupLogRepository(t *testing.T) {
	dbName := "test_repository.db"
	defer os.Remove(dbName)
	os.Remove(dbName)

	db, err := database.InitDB(dbName)
	require.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := database.NewGormBackupLogRepository(db)
	assert.NotNil(t, repo)

	t.Run("create backup logs", func(t *testing.T) {
		tests := []struct {
			name   string
			log    *models.BackupLog
			hasErr bool
		}{
			{
				name: "successful log",
				log: &models.BackupLog{
					Name:    "backup-test.zip",
					Size:    1024,
					Storage: "local",
					Status:  "success",
					Error:   "",
				},
				hasErr: false,
			},
			{
				name: "failed log with error",
				log: &models.BackupLog{
					Name:    "backup-failed.zip",
					Size:    2048,
					Storage: "s3",
					Status:  "failed",
					Error:   "connection timeout",
				},
				hasErr: false,
			},
			{
				name: "log without name",
				log: &models.BackupLog{
					Size:    100,
					Storage: "local",
					Status:  "success",
				},
				hasErr: false,
			},
			{
				name: "log with special characters",
				log: &models.BackupLog{
					Name:    "backup-2026-01-01_12-30-45.zip",
					Size:    1024,
					Storage: "local",
					Status:  "success",
				},
				hasErr: false,
			},
			{
				name: "log with empty status",
				log: &models.BackupLog{
					Name:    "empty-status.zip",
					Size:    100,
					Storage: "local",
					Status:  "",
				},
				hasErr: false,
			},
			{
				name: "log with large size",
				log: &models.BackupLog{
					Name:    "large-file.zip",
					Size:    1024 * 1024 * 1024, // 1GB
					Storage: "s3",
					Status:  "success",
				},
				hasErr: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := repo.CreateLog(tt.log)
				if tt.hasErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.NotZero(t, tt.log.ID)
					assert.False(t, tt.log.CreatedAt.IsZero())
				}
			})
		}

		// Проверяем, что все записи создались
		var count int64
		err := db.Model(&models.BackupLog{}).Count(&count).Error
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(6))
	})

	t.Run("create multiple logs and verify", func(t *testing.T) {
		// Создаем несколько логов через репозиторий
		logs := []*models.BackupLog{
			{Name: "multi1.zip", Size: 100, Storage: "local", Status: "success"},
			{Name: "multi2.zip", Size: 200, Storage: "s3", Status: "success"},
			{Name: "multi3.zip", Size: 300, Storage: "local", Status: "failed", Error: "error"},
		}

		for _, log := range logs {
			err := repo.CreateLog(log)
			assert.NoError(t, err)
			assert.NotZero(t, log.ID)
		}

		// Проверяем, что можно найти по имени
		var found models.BackupLog
		err := db.Where("name = ?", "multi2.zip").First(&found).Error
		assert.NoError(t, err)
		assert.Equal(t, "multi2.zip", found.Name)
		assert.Equal(t, int64(200), found.Size)
	})
}

func TestNewGormBackupLogRepository(t *testing.T) {
	dbName := "test_new_repository.db"
	defer os.Remove(dbName)
	os.Remove(dbName)

	db, err := database.InitDB(dbName)
	require.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	tests := []struct {
		name string
		db   *gorm.DB
	}{
		{
			name: "with valid db",
			db:   db,
		},
		{
			name: "with nil db",
			db:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := database.NewGormBackupLogRepository(tt.db)
			assert.NotNil(t, repo)
		})
	}
}
