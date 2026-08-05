package storage

import (
	"backup-service/internal/config"
	"context"
	"fmt"
	"log/slog"
	"os"
	"io"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const YandexS3 = "yandex_s3"

// Добавляем интерфейс для S3 клиента
//go:generate mockery
type S3Client interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

func init() {
	Register(YandexS3, func() (Storage, error) {
		cfg, _, err := config.NewYandexS3Config()
		if err != nil {
			return nil, fmt.Errorf("Неверная конфигурация для s3: %w", err)
		}
		return NewYandexS3Storage(cfg)
	})
}

type YandexS3Storage struct {
	client S3Client
	bucket string
}

func NewYandexS3Storage(cfg *config.YandexS3Config) (*YandexS3Storage, error) {
	// Создание клиента для Яндекс S3
	client := s3.New(s3.Options{
		Region: cfg.S3Region,
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
			cfg.S3Key,
			cfg.S3Secret,
			"",
		)),
		BaseEndpoint: aws.String(cfg.S3Endpoint),
	})

	return NewYandexS3StorageWithS3Client(client, cfg.S3Bucket)
}

func NewYandexS3StorageWithS3Client(client S3Client, bucket string) (*YandexS3Storage, error) {
	return &YandexS3Storage{
		client: client,
		bucket: bucket,
	}, nil
}




func (s *YandexS3Storage) Save(ctx context.Context, localPath string) (filename string, err error) {
	// Проверка существования файла
	if _, err := os.Stat(localPath); err != nil {
		return "", fmt.Errorf("Файл бэкапа не найден: %w", err)
	}

	// Открытие файла
	file, err := os.OpenFile(localPath, os.O_RDONLY, 0)
	if err != nil {
		return "", fmt.Errorf("Ошибка открытия файла %s: %w", localPath, err)
	}
	defer func() {
		if _, statErr := os.Stat(file.Name()); statErr == nil {
			if closeErr := file.Close(); closeErr != nil && err == nil {
				err = fmt.Errorf("Закрытие файла %s прервано: %w", localPath, closeErr)
			}
		}
	}()

	// Загрузка в S3
	filename = filepath.Base(localPath)
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filename),
		Body:   file,
	})
	if err != nil {
		return "", fmt.Errorf("Ошибка загрузки в S3: %w", err)
	}

	if err := file.Close(); err != nil {
		slog.Warn("Не удалось закрыть файл", "filepath", localPath, "error", err)
	}

	// Удаление локального файла
	if err := os.Remove(localPath); err != nil {
		slog.Warn("Не удалось удалить локальный файл", "filepath", localPath, "error", err)
	}

	return filename, nil
}




func (s *YandexS3Storage) List(ctx context.Context) ([]FileInfo, error) {
	result, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		return nil, fmt.Errorf("Ошибка получения списка файлов: %w", err)
	}

	var files []FileInfo
	for _, obj := range result.Contents {
		files = append(files, FileInfo{
			Name:      *obj.Key,
			Size:      *obj.Size,
			CreatedAt: *obj.LastModified,
		})
	}

	return files, nil
}




func (s *YandexS3Storage) Delete(ctx context.Context, path string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return fmt.Errorf("Ошибка удаления из S3: %w", err)
	}
	return nil
}



func (s *YandexS3Storage) Download(ctx context.Context, path string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return "", fmt.Errorf("Ошибка получения файла из S3: %w", err)
	}

	if result.Body == nil {
		return "", fmt.Errorf("Ошибка получения файла из S3: пустое тело файла")
	}
	defer result.Body.Close()

	tempFile, err := os.CreateTemp("", filepath.Base(path))
	if err != nil {
		return "", fmt.Errorf("Ошибка создания временного файла: %w", err)
	}

	tempPath := tempFile.Name()

	success := false
	defer func() {
		_ = tempFile.Close()

		if !success {
			_ = os.Remove(tempPath)
		}
	}()

	if _, err := io.Copy(tempFile, result.Body); err != nil {
		return "", fmt.Errorf("Ошибка сохранения файла: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return "", fmt.Errorf("Ошибка закрытия временного файла: %w", err)
	}

	success = true

	return tempPath, nil
}