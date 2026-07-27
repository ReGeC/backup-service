package storage

import (
	"backup-service/internal/config"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const YandexS3 StorageType = "yandex_s3"

func init() {
	cfg, enabled, err := config.NewYandexS3Config()
	if enabled {
		Register(YandexS3, func() (Storage, error) {
			if err != nil {
				return nil, fmt.Errorf("Неверная конфигурация для s3: %w", err)
			}
			return NewYandexS3Storage(cfg)
		})
	}
}

type YandexS3Storage struct {
	client *s3.Client
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

	return &YandexS3Storage{
		client: client,
		bucket: cfg.S3Bucket,
	}, nil
}




func (s *YandexS3Storage) Save(ctx context.Context, localPath string) (string, error) {
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
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("Закрытие файла %s прервано: %w", localPath, closeErr)
		}
	}()

	// Загрузка в S3
	filename := filepath.Base(localPath)
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key: aws.String(filename),
		Body: file,
	})
	if err != nil {
		return "", fmt.Errorf("Ошибка загрузки в S3: %w", err)
	}

	// Удаление локального файла
	if err := os.Remove(localPath); err != nil {
		log.Printf("WARNING: Не удалось удалить локальный файл %s: %v", localPath, err)
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
			Name: *obj.Key,
			Size: *obj.Size,
			CreatedAt: *obj.LastModified,
		})
	}

	return files, nil
}



func (s *YandexS3Storage) Delete(ctx context.Context, path string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key: aws.String(path),
	})
	if err != nil {
		return fmt.Errorf("Ошибка удаления из S3: %w", err)
	}
	return nil
}