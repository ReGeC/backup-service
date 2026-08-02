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