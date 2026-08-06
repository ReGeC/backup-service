package storage_test

import (
	"context"
	"testing"
	"time"

	"backup-service/internal/storage"
	mocks "backup-service/internal/storage/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetRegistry(t *testing.T) {
	t.Helper()

	storage.ResetRegistry()

	t.Cleanup(func() {
		storage.ResetRegistry()
	})
}

func TestRegister(t *testing.T) {
	resetRegistry(t)

	t.Run("register new storage type", func(t *testing.T) {
		factory := func() (storage.Storage, error) {
			return &mocks.MockStorage{}, nil
		}

		storage.Register("test", factory)

		// Проверяем, что регистрация прошла успешно
		s, err := storage.NewStorage("test")
		require.NoError(t, err)
		assert.NotNil(t, s)
	})

	t.Run("register multiple storage types", func(t *testing.T) {
		factory1 := func() (storage.Storage, error) {
			return &mocks.MockStorage{}, nil
		}
		factory2 := func() (storage.Storage, error) {
			return &mocks.MockStorage{}, nil
		}

		storage.Register("type1", factory1)
		storage.Register("type2", factory2)

		s1, err := storage.NewStorage("type1")
		require.NoError(t, err)
		assert.NotNil(t, s1)

		s2, err := storage.NewStorage("type2")
		require.NoError(t, err)
		assert.NotNil(t, s2)
	})

	t.Run("register overwrites existing type", func(t *testing.T) {
		factoryOld := func() (storage.Storage, error) {
			return &mocks.MockStorage{}, nil
		}
		factoryNew := func() (storage.Storage, error) {
			return &mocks.MockStorage{}, nil
		}

		storage.Register("same", factoryOld)
		storage.Register("same", factoryNew)

		s, err := storage.NewStorage("same")
		require.NoError(t, err)
		assert.NotNil(t, s)
	})
}

func TestResetRegistry(t *testing.T) {
	resetRegistry(t)

	t.Run("reset clears registry", func(t *testing.T) {
		// Регистрируем что-то
		factory := func() (storage.Storage, error) {
			return &mocks.MockStorage{}, nil
		}
		storage.Register("test", factory)

		// Проверяем, что тип существует
		_, err := storage.NewStorage("test")
		require.NoError(t, err)

		// Сбрасываем
		storage.ResetRegistry()

		// Проверяем, что тип больше не существует
		_, err = storage.NewStorage("test")
		require.Error(t, err)
		assert.EqualError(t, err, "неизвестный тип хранилища: test")
	})

	t.Run("reset empty registry", func(t *testing.T) {
		// Ресет пустого регистри
		storage.ResetRegistry()

		// Проверяем, что NewStorage возвращает ошибку для любого типа
		_, err := storage.NewStorage("any")
		require.Error(t, err)
		assert.EqualError(t, err, "неизвестный тип хранилища: any")
	})
}

func TestNewStorage(t *testing.T) {
	resetRegistry(t)

	t.Run("create storage for registered type", func(t *testing.T) {
		mockStorage := &mocks.MockStorage{}
		factory := func() (storage.Storage, error) {
			return mockStorage, nil
		}

		storage.Register("s3", factory)

		s, err := storage.NewStorage("s3")
		require.NoError(t, err)
		assert.Equal(t, mockStorage, s)
	})

	t.Run("create storage with factory returning error", func(t *testing.T) {
		expectedErr := assert.AnError
		factory := func() (storage.Storage, error) {
			return nil, expectedErr
		}

		storage.Register("failing", factory)

		s, err := storage.NewStorage("failing")
		require.Error(t, err)
		assert.Nil(t, s)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("create storage for unknown type returns error", func(t *testing.T) {
		s, err := storage.NewStorage("unknown")
		require.Error(t, err)
		assert.Nil(t, s)
		assert.EqualError(t, err, "неизвестный тип хранилища: unknown")
	})
}

func TestStorageInterface(t *testing.T) {
	t.Run("mock storage implements interface", func(t *testing.T) {
		ctx := context.Background()
		mockStorage := mocks.NewMockStorage(t)

		// Тестируем Save
		expectedPath := "test/path"
		expectedResult := "saved/file.txt"
		mockStorage.On("Save", ctx, expectedPath).Return(expectedResult, nil)

		result, err := mockStorage.Save(ctx, expectedPath)
		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)

		// Тестируем List
		expectedFiles := []storage.FileInfo{
			{Name: "file1.txt", Size: 100, CreatedAt: time.Now()},
			{Name: "file2.txt", Size: 200, CreatedAt: time.Now()},
		}
		mockStorage.On("List", ctx).Return(expectedFiles, nil)

		files, err := mockStorage.List(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expectedFiles, files)

		// Тестируем Delete
		expectedDeletePath := "test/delete"
		mockStorage.On("Delete", ctx, expectedDeletePath).Return(nil)

		err = mockStorage.Delete(ctx, expectedDeletePath)
		assert.NoError(t, err)

		mockStorage.AssertExpectations(t)
	})

	t.Run("mock storage with errors", func(t *testing.T) {
		ctx := context.Background()
		mockStorage := mocks.NewMockStorage(t)

		expectedErr := assert.AnError

		// Save with error
		mockStorage.On("Save", ctx, "error/path").Return("", expectedErr)
		result, err := mockStorage.Save(ctx, "error/path")
		assert.Error(t, err)
		assert.Empty(t, result)
		assert.Equal(t, expectedErr, err)

		// List with error
		mockStorage.On("List", ctx).Return([]storage.FileInfo(nil), expectedErr)
		files, err := mockStorage.List(ctx)
		assert.Error(t, err)
		assert.Nil(t, files)
		assert.Equal(t, expectedErr, err)

		// Delete with error
		mockStorage.On("Delete", ctx, "error/delete").Return(expectedErr)
		err = mockStorage.Delete(ctx, "error/delete")
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)

		mockStorage.AssertExpectations(t)
	})
}

func TestFileInfo(t *testing.T) {
	t.Run("create FileInfo struct", func(t *testing.T) {
		now := time.Now()
		info := storage.FileInfo{
			Name:      "test.txt",
			Size:      1024,
			CreatedAt: now,
		}

		assert.Equal(t, "test.txt", info.Name)
		assert.Equal(t, int64(1024), info.Size)
		assert.Equal(t, now, info.CreatedAt)
	})
}

func TestCleanupOldBackups(t *testing.T) {
	ctx := context.Background()

	t.Run("успешное удаление старых бэкапов", func(t *testing.T) {
		mockStorage := mocks.NewMockStorage(t)

		// Создаем файлы с разными датами (все старше 30 дней от текущей даты)
		now := time.Now()
		oldDate1 := now.AddDate(0, 0, -40)
		oldDate2 := now.AddDate(0, 0, -35)
		oldDate3 := now.AddDate(0, 0, -32)
		oldDate4 := now.AddDate(0, 0, -31)

		files := []storage.FileInfo{
			{Name: "db_" + oldDate1.Format("2006-01-02") + "_01-00_postgres.sql.gz", Size: 100, CreatedAt: oldDate1},
			{Name: "db_" + oldDate2.Format("2006-01-02") + "_01-00_postgres.sql.gz", Size: 100, CreatedAt: oldDate2},
			{Name: "db_" + oldDate3.Format("2006-01-02") + "_01-00_postgres.sql.gz", Size: 100, CreatedAt: oldDate3},
			{Name: "db_" + oldDate4.Format("2006-01-02") + "_01-00_postgres.sql.gz", Size: 100, CreatedAt: oldDate4},
		}

		mockStorage.On("List", ctx).Return(files, nil)

		// Ожидаем удаление ВСЕХ файлов (все старше 30 дней)
		for _, file := range files {
			mockStorage.On("Delete", ctx, file.Name).Return(nil)
		}

		deleted, err := storage.CleanupOldBackups(ctx, mockStorage, 30)

		assert.NoError(t, err)
		assert.Equal(t, 4, deleted)
		mockStorage.AssertExpectations(t)
	})

	t.Run("нет файлов для удаления", func(t *testing.T) {
		mockStorage := mocks.NewMockStorage(t)

		files := []storage.FileInfo{
			{Name: "db_2026-08-01_01-00_postgres.sql.gz", Size: 100, CreatedAt: time.Now()},
			{Name: "db_2026-08-02_01-00_postgres.sql.gz", Size: 100, CreatedAt: time.Now()},
			{Name: "db_2026-08-03_01-00_postgres.sql.gz", Size: 100, CreatedAt: time.Now()},
		}

		mockStorage.On("List", ctx).Return(files, nil)

		deleted, err := storage.CleanupOldBackups(ctx, mockStorage, 30)

		assert.NoError(t, err)
		assert.Equal(t, 0, deleted)
		mockStorage.AssertNotCalled(t, "Delete")
	})

	t.Run("пустой список файлов", func(t *testing.T) {
		mockStorage := mocks.NewMockStorage(t)

		mockStorage.On("List", ctx).Return([]storage.FileInfo{}, nil)

		deleted, err := storage.CleanupOldBackups(ctx, mockStorage, 30)

		assert.NoError(t, err)
		assert.Equal(t, 0, deleted)
		mockStorage.AssertExpectations(t)
	})

	t.Run("ошибка получения списка файлов", func(t *testing.T) {
		mockStorage := mocks.NewMockStorage(t)
		expectedErr := assert.AnError

		mockStorage.On("List", ctx).Return([]storage.FileInfo(nil), expectedErr)

		deleted, err := storage.CleanupOldBackups(ctx, mockStorage, 30)

		assert.Error(t, err)
		assert.Equal(t, 0, deleted)
		assert.Contains(t, err.Error(), "ошибка получения списка файлов")
		mockStorage.AssertExpectations(t)
	})

	t.Run("ошибка при удалении файла", func(t *testing.T) {
		mockStorage := mocks.NewMockStorage(t)

		files := []storage.FileInfo{
			{Name: "db_2026-01-01_01-00_postgres.sql.gz", Size: 100, CreatedAt: time.Now()},
			{Name: "db_2026-01-15_01-00_postgres.sql.gz", Size: 100, CreatedAt: time.Now()},
		}

		mockStorage.On("List", ctx).Return(files, nil)

		// Первый файл удаляется с ошибкой
		mockStorage.On("Delete", ctx, "db_2026-01-01_01-00_postgres.sql.gz").Return(assert.AnError)
		// Второй файл удаляется успешно
		mockStorage.On("Delete", ctx, "db_2026-01-15_01-00_postgres.sql.gz").Return(nil)

		deleted, err := storage.CleanupOldBackups(ctx, mockStorage, 30)

		assert.NoError(t, err)      // Ошибки удаления не должны останавливать процесс
		assert.Equal(t, 1, deleted) // Только второй файл удален успешно
		mockStorage.AssertExpectations(t)
	})

	t.Run("невалидный retentionDays", func(t *testing.T) {
		mockStorage := mocks.NewMockStorage(t)

		deleted, err := storage.CleanupOldBackups(ctx, mockStorage, 0)

		assert.Error(t, err)
		assert.Equal(t, 0, deleted)
		assert.EqualError(t, err, "retention days must be greater than 0")
		mockStorage.AssertNotCalled(t, "List")
	})

	t.Run("отмена контекста во время выполнения", func(t *testing.T) {
		mockStorage := mocks.NewMockStorage(t)
		ctx, cancel := context.WithCancel(context.Background())

		now := time.Now()
		oldDate1 := now.AddDate(0, 0, -40)
		oldDate2 := now.AddDate(0, 0, -35)
		oldDate3 := now.AddDate(0, 0, -32)

		files := []storage.FileInfo{
			{Name: "db_" + oldDate1.Format("2006-01-02") + "_01-00_postgres.sql.gz", Size: 100, CreatedAt: oldDate1},
			{Name: "db_" + oldDate2.Format("2006-01-02") + "_01-00_postgres.sql.gz", Size: 100, CreatedAt: oldDate2},
			{Name: "db_" + oldDate3.Format("2006-01-02") + "_01-00_postgres.sql.gz", Size: 100, CreatedAt: oldDate3},
		}

		mockStorage.On("List", ctx).Return(files, nil)

		// Отменяем контекст ДО вызова Delete
		cancel()

		// Не ожидаем вызовов Delete
		deleted, err := storage.CleanupOldBackups(ctx, mockStorage, 30)

		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
		assert.Equal(t, 0, deleted)
		mockStorage.AssertExpectations(t)
	})

	t.Run("файлы с неправильным форматом имени пропускаются", func(t *testing.T) {
		mockStorage := mocks.NewMockStorage(t)

		now := time.Now()
		oldDate := now.AddDate(0, 0, -40)   // старше 30 дней
		recentDate := now.AddDate(0, 0, -5) // моложе 30 дней

		files := []storage.FileInfo{
			{Name: "db_" + oldDate.Format("2006-01-02") + "_01-00_postgres.sql.gz", Size: 100, CreatedAt: oldDate},
			{Name: "invalid_filename.txt", Size: 100, CreatedAt: oldDate},
			{Name: "backup_" + recentDate.Format("2006-01-02") + ".tar.gz", Size: 100, CreatedAt: recentDate},
		}

		mockStorage.On("List", ctx).Return(files, nil)

		// Должен удалиться только первый файл (db_ с правильным форматом и старше 30 дней)
		mockStorage.On("Delete", ctx, files[0].Name).Return(nil)

		deleted, err := storage.CleanupOldBackups(ctx, mockStorage, 30)

		assert.NoError(t, err)
		assert.Equal(t, 1, deleted)
		mockStorage.AssertExpectations(t)
	})
}

func TestParseDateFromFilename(t *testing.T) {
	// Используем даты, которые точно старше 30 дней
	oldDate := time.Now().AddDate(0, 0, -40)
	recentDate := time.Now().AddDate(0, 0, -5)

	tests := []struct {
		name         string
		filename     string
		shouldDelete bool // ожидаем ли удаление
	}{
		{
			name:         "стандартный формат postgres",
			filename:     "db_" + oldDate.Format("2006-01-02") + "_06-56_postgres.sql.gz",
			shouldDelete: true,
		},
		{
			name:         "стандартный формат mysql",
			filename:     "db_" + oldDate.Format("2006-01-02") + "_06-56_mysql.sql.gz",
			shouldDelete: true,
		},
		{
			name:         "только дата без времени",
			filename:     "db_" + oldDate.Format("2006-01-02") + ".sql.gz",
			shouldDelete: true,
		},
		{
			name:         "файл не начинается с db_",
			filename:     "backup_" + oldDate.Format("2006-01-02") + ".tar.gz",
			shouldDelete: false,
		},
		{
			name:         "дата в начале",
			filename:     oldDate.Format("2006-01-02") + "_backup.sql.gz",
			shouldDelete: false,
		},
		{
			name:         "без даты",
			filename:     "backup.sql.gz",
			shouldDelete: false,
		},
		{
			name:         "пустое имя",
			filename:     "",
			shouldDelete: false,
		},
		{
			name:         "неправильный формат даты",
			filename:     "db_2026-08-5_06-56_postgres.sql.gz",
			shouldDelete: false,
		},
		{
			name:         "свежий файл (не старше 30 дней)",
			filename:     "db_" + recentDate.Format("2006-01-02") + "_06-56_postgres.sql.gz",
			shouldDelete: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := mocks.NewMockStorage(t)
			ctx := context.Background()

			files := []storage.FileInfo{
				{Name: tt.filename, Size: 100, CreatedAt: time.Now()},
			}

			mockStorage.On("List", ctx).Return(files, nil)

			if tt.shouldDelete {
				mockStorage.On("Delete", ctx, tt.filename).Return(nil)
			}

			deleted, err := storage.CleanupOldBackups(ctx, mockStorage, 30)

			assert.NoError(t, err)
			if tt.shouldDelete {
				assert.Equal(t, 1, deleted)
			} else {
				assert.Equal(t, 0, deleted)
			}
			mockStorage.AssertExpectations(t)
		})
	}
}

// Интеграционный тест с реальными датами
func TestCleanupOldBackups_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Пропускаем интеграционный тест в short режиме")
	}

	ctx := context.Background()
	mockStorage := mocks.NewMockStorage(t)

	// Создаем файлы с конкретными датами относительно сегодня
	now := time.Now()
	oldDate1 := now.AddDate(0, 0, -10)
	oldDate2 := now.AddDate(0, 0, -20)
	recentDate := now.AddDate(0, 0, -5)

	files := []storage.FileInfo{
		{
			Name:      "db_" + oldDate1.Format("2006-01-02") + "_01-00_postgres.sql.gz",
			Size:      100,
			CreatedAt: oldDate1,
		},
		{
			Name:      "db_" + oldDate2.Format("2006-01-02") + "_01-00_postgres.sql.gz",
			Size:      100,
			CreatedAt: oldDate2,
		},
		{
			Name:      "db_" + recentDate.Format("2006-01-02") + "_01-00_postgres.sql.gz",
			Size:      100,
			CreatedAt: recentDate,
		},
	}

	mockStorage.On("List", ctx).Return(files, nil)

	// Ожидаем удаление только файлов старше 7 дней
	mockStorage.On("Delete", ctx, files[0].Name).Return(nil)
	mockStorage.On("Delete", ctx, files[1].Name).Return(nil)
	// files[2] не должен удаляться

	deleted, err := storage.CleanupOldBackups(ctx, mockStorage, 7)

	assert.NoError(t, err)
	assert.Equal(t, 2, deleted)
	mockStorage.AssertExpectations(t)
}
