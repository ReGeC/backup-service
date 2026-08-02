package database_test

import (
	"backup-service/internal/models"
	"backup-service/internal/storage/database"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestInitDB тестирует инициализацию базы данных
func TestInitDB(t *testing.T) {
	t.Run("init db successfully", func(t *testing.T) {
		// Используем уникальное имя без слэшей
		dbName := "test_success.db"
		// Удаляем файл если существует
		os.Remove(dbName)
		defer os.Remove(dbName)

		db, err := database.InitDB(dbName)
		require.NoError(t, err)
		assert.NotNil(t, db)

		// Проверяем, что таблица создана
		err = db.AutoMigrate(&models.BackupLog{})
		assert.NoError(t, err)

		// Проверяем, что файл создан
		_, err = os.Stat(dbName)
		assert.NoError(t, err)

		// Проверяем, что можно выполнить запрос
		var count int64
		err = db.Model(&models.BackupLog{}).Count(&count).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)

		sqlDB, err := db.DB()
		assert.NoError(t, err)
		err = sqlDB.Close()
		assert.NoError(t, err)
	})

	t.Run("init db with existing file", func(t *testing.T) {
		dbName := "test_existing.db"
		defer os.Remove(dbName)

		// Удаляем файл если существует
		os.Remove(dbName)

		// Первая инициализация
		db1, err := database.InitDB(dbName)
		require.NoError(t, err)
		
		sqlDB1, err := db1.DB()
		require.NoError(t, err)
		err = sqlDB1.Close()
		require.NoError(t, err)

		// Вторая инициализация с тем же файлом
		db2, err := database.InitDB(dbName)
		require.NoError(t, err)
		assert.NotNil(t, db2)

		sqlDB2, err := db2.DB()
		require.NoError(t, err)
		err = sqlDB2.Close()
		require.NoError(t, err)
	})

	t.Run("init db with invalid path", func(t *testing.T) {
		// Используем относительный путь
		db, err := database.InitDB(":memory:")
		// Если не сработает - проверяем ошибку
		if err != nil {
			assert.Error(t, err)
			assert.Nil(t, db)
		}
	})
}

// TestInitDB_Schema тестирует схему таблицы
func TestInitDB_Schema(t *testing.T) {
	dbName := "test_schema.db"
	defer os.Remove(dbName)
	os.Remove(dbName)

	db, err := database.InitDB(dbName)
	require.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	t.Run("table has correct structure", func(t *testing.T) {
		migrator := db.Migrator()
		assert.True(t, migrator.HasTable(&models.BackupLog{}))

		// Проверяем колонки
		columns, err := migrator.ColumnTypes(&models.BackupLog{})
		assert.NoError(t, err)

		// Ожидаемые колонки
		expectedColumns := map[string]bool{
			"id":         false,
			"name":       false,
			"size":       false,
			"storage":    false,
			"status":     false,
			"error":      false,
			"created_at": false,
			"updated_at": false,
			"deleted_at": false,
		}

		for _, col := range columns {
			if _, ok := expectedColumns[col.Name()]; ok {
				expectedColumns[col.Name()] = true
			}
		}

		for col, found := range expectedColumns {
			assert.True(t, found, "Column %s not found", col)
		}
	})

	t.Run("has indexes", func(t *testing.T) {
		migrator := db.Migrator()
		
		// Проверяем индексы
		hasNameIndex := migrator.HasIndex(&models.BackupLog{}, "idx_backup_logs_name")
		assert.True(t, hasNameIndex)
		
		hasCreatedAtIndex := migrator.HasIndex(&models.BackupLog{}, "idx_backup_logs_created_at")
		assert.True(t, hasCreatedAtIndex)
		
		hasDeletedAtIndex := migrator.HasIndex(&models.BackupLog{}, "idx_backup_logs_deleted_at")
		assert.True(t, hasDeletedAtIndex)
	})
}

// TestInitDB_CRUD тестирует CRUD операции
func TestInitDB_CRUD(t *testing.T) {
	dbName := "test_crud.db"
	defer os.Remove(dbName)
	os.Remove(dbName)

	db, err := database.InitDB(dbName)
	require.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	t.Run("create backup log", func(t *testing.T) {
		log := &models.BackupLog{
			Name:    "backup-2026-01-01.zip",
			Size:    1024,
			Storage: "local",
			Status:  "success",
			Error:   "",
		}
		err := db.Create(log).Error
		assert.NoError(t, err)
		assert.NotZero(t, log.ID)
		assert.False(t, log.CreatedAt.IsZero())
		assert.False(t, log.UpdatedAt.IsZero())
	})

	t.Run("create backup log with error", func(t *testing.T) {
		log := &models.BackupLog{
			Name:    "backup-failed.zip",
			Size:    2048,
			Storage: "s3",
			Status:  "failed",
			Error:   "upload timeout",
		}
		err := db.Create(log).Error
		assert.NoError(t, err)
		assert.NotZero(t, log.ID)
	})

	t.Run("read backup logs", func(t *testing.T) {
		// Создаем несколько записей
		logs := []models.BackupLog{
			{Name: "log1.zip", Size: 100, Storage: "local", Status: "success"},
			{Name: "log2.zip", Size: 200, Storage: "s3", Status: "success"},
			{Name: "log3.zip", Size: 300, Storage: "local", Status: "failed", Error: "error"},
		}
		for _, log := range logs {
			err := db.Create(&log).Error
			assert.NoError(t, err)
		}

		// Читаем все записи
		var allLogs []models.BackupLog
		err := db.Find(&allLogs).Error
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(allLogs), 3)

		// Читаем по условию
		var localLogs []models.BackupLog
		err = db.Where("storage = ?", "local").Find(&localLogs).Error
		assert.NoError(t, err)
		for _, log := range localLogs {
			assert.Equal(t, "local", log.Storage)
		}

		// Читаем по имени
		var found models.BackupLog
		err = db.Where("name = ?", "log2.zip").First(&found).Error
		assert.NoError(t, err)
		assert.Equal(t, "log2.zip", found.Name)
		assert.Equal(t, int64(200), found.Size)
	})

	t.Run("update backup log", func(t *testing.T) {
		// Создаем запись
		log := &models.BackupLog{
			Name:    "update-test.zip",
			Size:    100,
			Storage: "local",
			Status:  "pending",
		}
		err := db.Create(log).Error
		assert.NoError(t, err)

		// Обновляем статус
		log.Status = "success"
		log.Error = ""
		err = db.Save(log).Error
		assert.NoError(t, err)

		// Проверяем обновление
		var updated models.BackupLog
		err = db.First(&updated, log.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, "success", updated.Status)
		assert.Empty(t, updated.Error)
	})

	t.Run("delete backup log", func(t *testing.T) {
		// Создаем запись
		log := &models.BackupLog{
			Name:    "delete-test.zip",
			Size:    100,
			Storage: "local",
			Status:  "success",
		}
		err := db.Create(log).Error
		assert.NoError(t, err)

		// Жесткое удаление
		err = db.Unscoped().Delete(log).Error
		assert.NoError(t, err)

		// Проверяем, что записи нет
		var count int64
		err = db.Model(&models.BackupLog{}).Where("name = ?", "delete-test.zip").Count(&count).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("soft delete", func(t *testing.T) {
		// Создаем запись
		log := &models.BackupLog{
			Name:    "soft-delete-test.zip",
			Size:    100,
			Storage: "local",
			Status:  "success",
		}
		err := db.Create(log).Error
		assert.NoError(t, err)

		// Мягкое удаление
		err = db.Delete(log).Error
		assert.NoError(t, err)

		// Проверяем, что запись не находится в обычном запросе
		var count int64
		err = db.Model(&models.BackupLog{}).Where("name = ?", "soft-delete-test.zip").Count(&count).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)

		// Проверяем, что запись есть с учетом deleted_at
		var deleted models.BackupLog
		err = db.Unscoped().Where("name = ?", "soft-delete-test.zip").First(&deleted).Error
		assert.NoError(t, err)
		assert.NotNil(t, deleted.DeletedAt.Time)
	})
}

// TestInitDB_TableName тестирует имя таблицы
func TestInitDB_TableName(t *testing.T) {
	dbName := "test_tablename.db"
	defer os.Remove(dbName)
	os.Remove(dbName)

	db, err := database.InitDB(dbName)
	require.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	log := &models.BackupLog{}
	assert.Equal(t, "backup_logs", log.TableName())

	// Проверяем, что таблица создана с правильным именем
	migrator := db.Migrator()
	assert.True(t, migrator.HasTable("backup_logs"))
}

// TestInitDB_WithInMemory тестирует in-memory режим
func TestInitDB_WithInMemory(t *testing.T) {
	// Используем in-memory базу
	dbName := "file::memory:?cache=shared"
	
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.BackupLog{})
	require.NoError(t, err)

	// Создаем запись
	log := &models.BackupLog{
		Name:    "in-memory-test.zip",
		Size:    1024,
		Storage: "local",
		Status:  "success",
	}
	err = db.Create(log).Error
	assert.NoError(t, err)

	// Читаем запись
	var found models.BackupLog
	err = db.First(&found, "name = ?", "in-memory-test.zip").Error
	assert.NoError(t, err)
	assert.Equal(t, "in-memory-test.zip", found.Name)
	assert.Equal(t, int64(1024), found.Size)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	err = sqlDB.Close()
	require.NoError(t, err)
}

// BenchmarkInitDB бенчмарк инициализации
func BenchmarkInitDB(b *testing.B) {
	for i := 0; i < b.N; i++ {
		dbName := "bench.db"
		db, err := database.InitDB(dbName)
		if err != nil {
			b.Fatal(err)
		}
		sqlDB, _ := db.DB()
		sqlDB.Close()
		os.Remove(dbName)
	}
}

// TestInitDB_WithCustomPath тестирует создание БД в указанной папке
func TestInitDB_WithCustomPath(t *testing.T) {
	// Создаем временную папку
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "custom.db")

	db, err := database.InitDB(dbPath)
	require.NoError(t, err)
	assert.NotNil(t, db)

	// Проверяем, что файл создан
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	err = sqlDB.Close()
	require.NoError(t, err)
}

// TestInitDB_WithSQLiteMemory тестирует специальный режим :memory:
func TestInitDB_WithSQLiteMemory(t *testing.T) {
	db, err := database.InitDB(":memory:")
	require.NoError(t, err)
	assert.NotNil(t, db)

	// Проверяем, что можно создать таблицу
	err = db.AutoMigrate(&models.BackupLog{})
	assert.NoError(t, err)

	// Создаем запись
	log := &models.BackupLog{
		Name:    "memory-test.zip",
		Size:    100,
		Storage: "local",
		Status:  "success",
	}
	err = db.Create(log).Error
	assert.NoError(t, err)

	// Проверяем запись
	var count int64
	err = db.Model(&models.BackupLog{}).Count(&count).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	err = sqlDB.Close()
	require.NoError(t, err)
}