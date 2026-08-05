package storage_test

import (
	"backup-service/internal/config"
	"backup-service/internal/storage"
	mocks "backup-service/internal/storage/mocks"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestNewYandexS3Storage тестирует создание хранилища
func TestNewYandexS3Storage(t *testing.T) {
	t.Run("create storage with valid config", func(t *testing.T) {
		cfg := &config.YandexS3Config{
			S3Key:      "test-key",
			S3Secret:   "test-secret",
			S3Region:   "ru-central1",
			S3Endpoint: "https://storage.yandexcloud.net",
			S3Bucket:   "test-bucket",
		}

		s, err := storage.NewYandexS3Storage(cfg)
		require.NoError(t, err)
		assert.NotNil(t, s)
	})

	t.Run("create storage with empty config", func(t *testing.T) {
		cfg := &config.YandexS3Config{
			S3Bucket: "test-bucket",
		}

		s, err := storage.NewYandexS3Storage(cfg)
		require.NoError(t, err)
		assert.NotNil(t, s)
	})
}

// TestNewYandexS3StorageWithS3Client тестирует создание с кастомным клиентом
func TestNewYandexS3StorageWithS3Client(t *testing.T) {
	t.Run("create with mock client", func(t *testing.T) {
		mockClient := mocks.NewMockS3Client(t)
		s, err := storage.NewYandexS3StorageWithS3Client(mockClient, "test-bucket")
		require.NoError(t, err)
		assert.NotNil(t, s)
	})
}

// TestYandexS3Storage_Save тестирует метод Save
func TestYandexS3Storage_Save(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	t.Run("save file successfully", func(t *testing.T) {
		// Создаем тестовый файл
		testFile := filepath.Join(tempDir, "backup.zip")
		err := os.WriteFile(testFile, []byte("test data"), 0644)
		require.NoError(t, err)

		// Создаем мок клиента
		mockClient := mocks.NewMockS3Client(t)
		s, err := storage.NewYandexS3StorageWithS3Client(mockClient, "test-bucket")
		require.NoError(t, err)

		// Настраиваем ожидания
		mockClient.On("PutObject", ctx, mock.MatchedBy(func(input *s3.PutObjectInput) bool {
			return aws.ToString(input.Bucket) == "test-bucket" &&
				aws.ToString(input.Key) == "backup.zip"
		})).Return(&s3.PutObjectOutput{}, nil)

		// Выполняем сохранение
		filename, err := s.Save(ctx, testFile)
		require.NoError(t, err)
		assert.Equal(t, "backup.zip", filename)

		// Проверяем, что локальный файл удален
		_, err = os.Stat(testFile)
		assert.True(t, os.IsNotExist(err))

		mockClient.AssertExpectations(t)
	})

	t.Run("save file with custom filename", func(t *testing.T) {
		// Создаем тестовый файл с другим именем
		testFile := filepath.Join(tempDir, "my-backup-2026.zip")
		err := os.WriteFile(testFile, []byte("test data"), 0644)
		require.NoError(t, err)

		mockClient := mocks.NewMockS3Client(t)
		s, err := storage.NewYandexS3StorageWithS3Client(mockClient, "test-bucket")
		require.NoError(t, err)

		mockClient.On("PutObject", ctx, mock.MatchedBy(func(input *s3.PutObjectInput) bool {
			return aws.ToString(input.Bucket) == "test-bucket" &&
				aws.ToString(input.Key) == "my-backup-2026.zip"
		})).Return(&s3.PutObjectOutput{}, nil)

		filename, err := s.Save(ctx, testFile)
		require.NoError(t, err)
		assert.Equal(t, "my-backup-2026.zip", filename)

		mockClient.AssertExpectations(t)
	})

	t.Run("save file not found", func(t *testing.T) {
		mockClient := mocks.NewMockS3Client(t)
		s, err := storage.NewYandexS3StorageWithS3Client(mockClient, "test-bucket")
		require.NoError(t, err)

		nonExistentFile := filepath.Join(tempDir, "non-existent.zip")
		filename, err := s.Save(ctx, nonExistentFile)

		require.Error(t, err)
		assert.Empty(t, filename)
		assert.ErrorContains(t, err, "Файл бэкапа не найден")
	})

	t.Run("save file with s3 upload error", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "backup-error.zip")
		err := os.WriteFile(testFile, []byte("test data"), 0644)
		require.NoError(t, err)

		mockClient := mocks.NewMockS3Client(t)
		s, err := storage.NewYandexS3StorageWithS3Client(mockClient, "test-bucket")
		require.NoError(t, err)

		expectedErr := errors.New("RequestError: send request failed")
		mockClient.On("PutObject", ctx, mock.Anything).Return(nil, expectedErr)

		filename, err := s.Save(ctx, testFile)

		require.Error(t, err)
		assert.Empty(t, filename)
		assert.ErrorContains(t, err, "Ошибка загрузки в S3")
		assert.ErrorContains(t, err, expectedErr.Error())

		// Проверяем, что локальный файл НЕ был удален при ошибке
		_, err = os.Stat(testFile)
		assert.NoError(t, err)

		mockClient.AssertExpectations(t)
	})

	t.Run("save file with s3 access denied", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "backup-denied.zip")
		err := os.WriteFile(testFile, []byte("test data"), 0644)
		require.NoError(t, err)

		mockClient := mocks.NewMockS3Client(t)
		s, err := storage.NewYandexS3StorageWithS3Client(mockClient, "test-bucket")
		require.NoError(t, err)

		expectedErr := errors.New("AccessDenied: Access Denied")
		mockClient.On("PutObject", ctx, mock.Anything).Return(nil, expectedErr)

		filename, err := s.Save(ctx, testFile)

		require.Error(t, err)
		assert.Empty(t, filename)
		assert.ErrorContains(t, err, "Ошибка загрузки в S3")
		assert.ErrorContains(t, err, expectedErr.Error())

		mockClient.AssertExpectations(t)
	})

	t.Run("save file with empty bucket", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "backup-empty-bucket.zip")
		err := os.WriteFile(testFile, []byte("test data"), 0644)
		require.NoError(t, err)

		mockClient := mocks.NewMockS3Client(t)
		s, err := storage.NewYandexS3StorageWithS3Client(mockClient, "")
		require.NoError(t, err)

		mockClient.On("PutObject", ctx, mock.MatchedBy(func(input *s3.PutObjectInput) bool {
			return aws.ToString(input.Bucket) == ""
		})).Return(nil, errors.New("bucket is empty"))

		filename, err := s.Save(ctx, testFile)

		require.Error(t, err)
		assert.Empty(t, filename)
		assert.ErrorContains(t, err, "Ошибка загрузки в S3")

		mockClient.AssertExpectations(t)
	})
}

// TestYandexS3Storage_List тестирует метод List
func TestYandexS3Storage_List(t *testing.T) {
	ctx := context.Background()

	t.Run("list files successfully", func(t *testing.T) {
		now := time.Now()
		mockObjects := []types.Object{
			{
				Key:          aws.String("backup-2026-01-01.zip"),
				Size:         aws.Int64(1024),
				LastModified: &now,
			},
			{
				Key:          aws.String("backup-2026-01-02.zip"),
				Size:         aws.Int64(2048),
				LastModified: &now,
			},
			{
				Key:          aws.String("backup-2026-01-03.zip"),
				Size:         aws.Int64(3072),
				LastModified: &now,
			},
		}

		mockClient := mocks.NewMockS3Client(t)
		s, err := storage.NewYandexS3StorageWithS3Client(mockClient, "test-bucket")
		require.NoError(t, err)

		mockClient.On("ListObjectsV2", ctx, &s3.ListObjectsV2Input{
			Bucket: aws.String("test-bucket"),
		}).Return(&s3.ListObjectsV2Output{
			Contents: mockObjects,
		}, nil)

		files, err := s.List(ctx)

		require.NoError(t, err)
		assert.Len(t, files, 3)
		assert.Equal(t, "backup-2026-01-01.zip", files[0].Name)
		assert.Equal(t, int64(1024), files[0].Size)
		assert.Equal(t, now, files[0].CreatedAt)
		assert.Equal(t, "backup-2026-01-02.zip", files[1].Name)
		assert.Equal(t, int64(2048), files[1].Size)
		assert.Equal(t, "backup-2026-01-03.zip", files[2].Name)
		assert.Equal(t, int64(3072), files[2].Size)

		mockClient.AssertExpectations(t)
	})

	t.Run("list files with empty bucket", func(t *testing.T) {
		mockClient := mocks.NewMockS3Client(t)
		s, err := storage.NewYandexS3StorageWithS3Client(mockClient, "test-bucket")
		require.NoError(t, err)

		mockClient.On("ListObjectsV2", ctx, &s3.ListObjectsV2Input{
			Bucket: aws.String("test-bucket"),
		}).Return(&s3.ListObjectsV2Output{
			Contents: []types.Object{},
		}, nil)

		files, err := s.List(ctx)

		require.NoError(t, err)
		assert.Empty(t, files)

		mockClient.AssertExpectations(t)
	})

	t.Run("list files with s3 error", func(t *testing.T) {
		mockClient := mocks.NewMockS3Client(t)
		s, err := storage.NewYandexS3StorageWithS3Client(mockClient, "test-bucket")
		require.NoError(t, err)

		expectedErr := errors.New("NoSuchBucket: The specified bucket does not exist")
		mockClient.On("ListObjectsV2", ctx, &s3.ListObjectsV2Input{
			Bucket: aws.String("test-bucket"),
		}).Return(nil, expectedErr)

		files, err := s.List(ctx)

		require.Error(t, err)
		assert.Nil(t, files)
		assert.ErrorContains(t, err, "Ошибка получения списка файлов")
		assert.ErrorContains(t, err, expectedErr.Error())

		mockClient.AssertExpectations(t)
	})

	t.Run("list files with access denied", func(t *testing.T) {
		mockClient := mocks.NewMockS3Client(t)
		s, err := storage.NewYandexS3StorageWithS3Client(mockClient, "test-bucket")
		require.NoError(t, err)

		expectedErr := errors.New("AccessDenied: Access Denied")
		mockClient.On("ListObjectsV2", ctx, &s3.ListObjectsV2Input{
			Bucket: aws.String("test-bucket"),
		}).Return(nil, expectedErr)

		files, err := s.List(ctx)

		require.Error(t, err)
		assert.Nil(t, files)
		assert.ErrorContains(t, err, "Ошибка получения списка файлов")

		mockClient.AssertExpectations(t)
	})
}

// TestYandexS3Storage_Delete тестирует метод Delete
func TestYandexS3Storage_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("delete file successfully", func(t *testing.T) {
		mockClient := mocks.NewMockS3Client(t)
		s, err := storage.NewYandexS3StorageWithS3Client(mockClient, "test-bucket")
		require.NoError(t, err)

		path := "backup-to-delete.zip"
		mockClient.On("DeleteObject", ctx, &s3.DeleteObjectInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String(path),
		}).Return(&s3.DeleteObjectOutput{}, nil)

		err = s.Delete(ctx, path)
		require.NoError(t, err)

		mockClient.AssertExpectations(t)
	})

	t.Run("delete file with s3 error", func(t *testing.T) {
		mockClient := mocks.NewMockS3Client(t)
		s, err := storage.NewYandexS3StorageWithS3Client(mockClient, "test-bucket")
		require.NoError(t, err)

		path := "non-existent-file.zip"
		expectedErr := errors.New("NoSuchKey: The specified key does not exist")
		mockClient.On("DeleteObject", ctx, &s3.DeleteObjectInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String(path),
		}).Return(nil, expectedErr)

		err = s.Delete(ctx, path)

		require.Error(t, err)
		assert.ErrorContains(t, err, "Ошибка удаления из S3")
		assert.ErrorContains(t, err, expectedErr.Error())

		mockClient.AssertExpectations(t)
	})

	t.Run("delete file with access denied", func(t *testing.T) {
		mockClient := mocks.NewMockS3Client(t)
		s, err := storage.NewYandexS3StorageWithS3Client(mockClient, "test-bucket")
		require.NoError(t, err)

		path := "protected-file.zip"
		expectedErr := errors.New("AccessDenied: Access Denied")
		mockClient.On("DeleteObject", ctx, &s3.DeleteObjectInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String(path),
		}).Return(nil, expectedErr)

		err = s.Delete(ctx, path)

		require.Error(t, err)
		assert.ErrorContains(t, err, "Ошибка удаления из S3")

		mockClient.AssertExpectations(t)
	})

	t.Run("delete file with empty bucket", func(t *testing.T) {
		mockClient := mocks.NewMockS3Client(t)
		s, err := storage.NewYandexS3StorageWithS3Client(mockClient, "")
		require.NoError(t, err)

		path := "file.zip"
		mockClient.On("DeleteObject", ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(""),
			Key:    aws.String(path),
		}).Return(nil, errors.New("bucket is empty"))

		err = s.Delete(ctx, path)

		require.Error(t, err)
		assert.ErrorContains(t, err, "Ошибка удаления из S3")

		mockClient.AssertExpectations(t)
	})
}

// TestYandexS3Storage_IntegrationStyle тесты с реальным сценарием
// TestYandexS3Storage_FullLifecycle тест полного жизненного цикла
func TestYandexS3Storage_FullLifecycle(t *testing.T) {
	ctx := context.Background()

	t.Run("full lifecycle: save -> list -> delete", func(t *testing.T) {
		tempDir := t.TempDir()
		mockClient := mocks.NewMockS3Client(t)
		s, err := storage.NewYandexS3StorageWithS3Client(mockClient, "test-bucket")
		require.NoError(t, err)

		// 1. Сохраняем файл
		testFile := filepath.Join(tempDir, "lifecycle-backup.zip")
		err = os.WriteFile(testFile, []byte("test data for lifecycle"), 0644)
		require.NoError(t, err)

		mockClient.On("PutObject", ctx, mock.MatchedBy(func(input *s3.PutObjectInput) bool {
			return aws.ToString(input.Key) == "lifecycle-backup.zip"
		})).Return(&s3.PutObjectOutput{}, nil)

		filename, err := s.Save(ctx, testFile)
		require.NoError(t, err)
		assert.Equal(t, "lifecycle-backup.zip", filename)

		// 2. Получаем список файлов (должен быть 1 файл)
		now := time.Now()
		mockObjects := []types.Object{
			{
				Key:          aws.String("lifecycle-backup.zip"),
				Size:         aws.Int64(1024),
				LastModified: &now,
			},
		}

		// Первый вызов ListObjectsV2 - возвращаем файл
		mockClient.On("ListObjectsV2", ctx, &s3.ListObjectsV2Input{
			Bucket: aws.String("test-bucket"),
		}).Return(&s3.ListObjectsV2Output{
			Contents: mockObjects,
		}, nil).Once()

		files, err := s.List(ctx)
		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.Equal(t, "lifecycle-backup.zip", files[0].Name)

		// 3. Удаляем файл
		mockClient.On("DeleteObject", ctx, &s3.DeleteObjectInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("lifecycle-backup.zip"),
		}).Return(&s3.DeleteObjectOutput{}, nil).Once()

		err = s.Delete(ctx, "lifecycle-backup.zip")
		require.NoError(t, err)

		// 4. Проверяем, что файла больше нет в списке
		// Второй вызов ListObjectsV2 - возвращаем пустой список
		mockClient.On("ListObjectsV2", ctx, &s3.ListObjectsV2Input{
			Bucket: aws.String("test-bucket"),
		}).Return(&s3.ListObjectsV2Output{
			Contents: []types.Object{},
		}, nil).Once()

		files, err = s.List(ctx)
		require.NoError(t, err)
		assert.Empty(t, files)

		mockClient.AssertExpectations(t)
	})
}
// TestYandexS3Storage_Errors тестирует обработку ошибок
// TestYandexS3Storage_Errors тестирует обработку ошибок
func TestYandexS3Storage_Errors(t *testing.T) {
	ctx := context.Background()

	t.Run("multiple error types on save", func(t *testing.T) {
		tempDir := t.TempDir()

		errorTypes := []struct {
			name     string
			err      error
			contains string
		}{
			{
				name:     "network error",
				err:      errors.New("connection refused"),
				contains: "connection refused",
			},
			{
				name:     "timeout error",
				err:      errors.New("timeout: context deadline exceeded"),
				contains: "timeout",
			},
			{
				name:     "size limit error",
				err:      errors.New("EntityTooLarge: Your proposed upload exceeds the maximum allowed object size"),
				contains: "EntityTooLarge",
			},
		}

		for _, tt := range errorTypes {
			t.Run(tt.name, func(t *testing.T) {
				testFile := filepath.Join(tempDir, tt.name+"-backup.zip")
				err := os.WriteFile(testFile, []byte("test data"), 0644)
				require.NoError(t, err)

				mockClient := mocks.NewMockS3Client(t)
				s, err := storage.NewYandexS3StorageWithS3Client(mockClient, "test-bucket")
				require.NoError(t, err)

				mockClient.On("PutObject", ctx, mock.Anything).Return(nil, tt.err)

				filename, err := s.Save(ctx, testFile)
				require.Error(t, err)
				assert.Empty(t, filename)
				assert.ErrorContains(t, err, "Ошибка загрузки в S3")
				assert.ErrorContains(t, err, tt.contains)

				mockClient.AssertExpectations(t)
			})
		}
	})

	t.Run("error messages are preserved for List", func(t *testing.T) {
		s3Errors := []struct {
			name string
			err  error
		}{
			{
				name: "no such bucket",
				err:  errors.New("NoSuchBucket: The specified bucket does not exist"),
			},
			{
				name: "no such key",
				err:  errors.New("NoSuchKey: The specified key does not exist"),
			},
			{
				name: "access denied",
				err:  errors.New("AccessDenied: Access Denied"),
			},
		}

		for _, tt := range s3Errors {
			t.Run(tt.name, func(t *testing.T) {
				mockClient := mocks.NewMockS3Client(t)
				s, err := storage.NewYandexS3StorageWithS3Client(mockClient, "test-bucket")
				require.NoError(t, err)

				// Настраиваем мок для конкретной ошибки
				mockClient.On("ListObjectsV2", ctx, &s3.ListObjectsV2Input{
					Bucket: aws.String("test-bucket"),
				}).Return(nil, tt.err)

				files, err := s.List(ctx)
				require.Error(t, err)
				assert.Nil(t, files)
				assert.ErrorContains(t, err, "Ошибка получения списка файлов")
				assert.ErrorContains(t, err, tt.err.Error())

				mockClient.AssertExpectations(t)
			})
		}
	})

	t.Run("error messages are preserved for Delete", func(t *testing.T) {
		s3Errors := []struct {
			name string
			err  error
		}{
			{
				name: "no such key",
				err:  errors.New("NoSuchKey: The specified key does not exist"),
			},
			{
				name: "access denied",
				err:  errors.New("AccessDenied: Access Denied"),
			},
		}

		for _, tt := range s3Errors {
			t.Run(tt.name, func(t *testing.T) {
				mockClient := mocks.NewMockS3Client(t)
				s, err := storage.NewYandexS3StorageWithS3Client(mockClient, "test-bucket")
				require.NoError(t, err)

				path := "test-file.zip"
				mockClient.On("DeleteObject", ctx, &s3.DeleteObjectInput{
					Bucket: aws.String("test-bucket"),
					Key:    aws.String(path),
				}).Return(nil, tt.err)

				err = s.Delete(ctx, path)
				require.Error(t, err)
				assert.ErrorContains(t, err, "Ошибка удаления из S3")
				assert.ErrorContains(t, err, tt.err.Error())

				mockClient.AssertExpectations(t)
			})
		}
	})
}

type errorReader struct{}

func (e *errorReader) Read(p []byte) (int, error) {
	return 0, errors.New("read error")
}

// TestYandexS3Storage_Download тестирует метод Download
func TestYandexS3Storage_Download(t *testing.T) {
	ctx := context.Background()

	t.Run("download file successfully", func(t *testing.T) {
		mockClient := mocks.NewMockS3Client(t)

		s, err := storage.NewYandexS3StorageWithS3Client(mockClient, "test-bucket")
		require.NoError(t, err)

		content := "test backup content"

		mockClient.On("GetObject", ctx, mock.MatchedBy(func(input *s3.GetObjectInput) bool {
			return aws.ToString(input.Bucket) == "test-bucket" &&
				aws.ToString(input.Key) == "backup.zip"
		})).Return(&s3.GetObjectOutput{
			Body: io.NopCloser(strings.NewReader(content)),
		}, nil)

		path, err := s.Download(ctx, "backup.zip")

		require.NoError(t, err)
		assert.NotEmpty(t, path)

		defer os.Remove(path)

		data, err := os.ReadFile(path)
		require.NoError(t, err)

		assert.Equal(t, content, string(data))

		mockClient.AssertExpectations(t)
	})


	t.Run("download file not found", func(t *testing.T) {
		mockClient := mocks.NewMockS3Client(t)

		s, err := storage.NewYandexS3StorageWithS3Client(mockClient, "test-bucket")
		require.NoError(t, err)

		expectedErr := errors.New("NoSuchKey: The specified key does not exist")

		mockClient.On("GetObject", ctx, mock.Anything).
			Return(nil, expectedErr)

		path, err := s.Download(ctx, "missing.zip")

		assert.Error(t, err)
		assert.Empty(t, path)

		assert.ErrorContains(t, err, "Ошибка получения файла из S3")
		assert.ErrorContains(t, err, expectedErr.Error())

		mockClient.AssertExpectations(t)
	})


	t.Run("download with empty body", func(t *testing.T) {
		mockClient := mocks.NewMockS3Client(t)

		s, err := storage.NewYandexS3StorageWithS3Client(mockClient, "test-bucket")
		require.NoError(t, err)

		mockClient.On("GetObject", ctx, mock.Anything).
			Return(&s3.GetObjectOutput{
				Body: nil,
			}, nil)

		path, err := s.Download(ctx, "empty.zip")

		assert.Error(t, err)
		assert.Empty(t, path)

		assert.ErrorContains(t, err, "пустое тело файла")

		mockClient.AssertExpectations(t)
	})


	t.Run("download with copy error", func(t *testing.T) {
		mockClient := mocks.NewMockS3Client(t)

		s, err := storage.NewYandexS3StorageWithS3Client(mockClient, "test-bucket")
		require.NoError(t, err)

		mockClient.On("GetObject", ctx, mock.Anything).
			Return(&s3.GetObjectOutput{
				Body: io.NopCloser(&errorReader{}),
			}, nil)

		path, err := s.Download(ctx, "broken.zip")

		assert.Error(t, err)
		assert.Empty(t, path)

		assert.ErrorContains(t, err, "Ошибка сохранения файла")

		mockClient.AssertExpectations(t)
	})


	t.Run("download with canceled context", func(t *testing.T) {
		mockClient := mocks.NewMockS3Client(t)

		s, err := storage.NewYandexS3StorageWithS3Client(mockClient, "test-bucket")
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		path, err := s.Download(ctx, "backup.zip")

		assert.Error(t, err)
		assert.Empty(t, path)
		assert.Equal(t, context.Canceled, err)

		mockClient.AssertNotCalled(t, "GetObject")
	})
}