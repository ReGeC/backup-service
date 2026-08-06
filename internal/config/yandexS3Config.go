package config

import (
	"errors"
	"strings"
)

var (
	ErrEmptyS3Endpoint   = errors.New("yandex_s3 endpoint is empty")
	ErrInvalidS3Endpoint = errors.New("yandex_s3 endpoint invalid")
	ErrEmptyS3Bucket     = errors.New("yandex_s3 bucket is empty")
	ErrEmptyS3Region     = errors.New("yandex_s3 region is empty")
	ErrEmptyS3Key        = errors.New("yandex_s3 key is empty")
	ErrEmptyS3Secret     = errors.New("yandex_s3 secret is empty")
)

type YandexS3Config struct {
	loader ConfigLoader

	S3Endpoint string
	S3Bucket   string
	S3Region   string
	S3Key      string
	S3Secret   string
}

func (s *YandexS3Config) LoadConfig() (bool, error) {
	// Не добаляем Enable, так как хранилище выбирается только одно
	// а не подключается как модуль
	// Оставлено конструкция как у других конфигов на будущее, если вдруг потребуется
	s.S3Endpoint = s.loader.GetString([]string{"s3", "yandex", "endpoint"}, "https://storage.yandexcloud.net")
	s.S3Bucket = s.loader.GetString([]string{"s3", "yandex", "bucket"}, "")
	s.S3Region = s.loader.GetString([]string{"s3", "yandex", "region"}, "ru-central1")
	s.S3Key = s.loader.GetString([]string{"s3", "yandex", "key"}, "")
	s.S3Secret = s.loader.GetString([]string{"s3", "yandex", "secret"}, "")

	return true, s.ValidateConfig()
}

func (s *YandexS3Config) ValidateConfig() error {
	if s.S3Endpoint == "" {
		return ErrEmptyS3Endpoint
	}
	if !strings.HasPrefix(s.S3Endpoint, "http://") && !strings.HasPrefix(s.S3Endpoint, "https://") {
		return ErrInvalidS3Endpoint
	}
	if s.S3Bucket == "" {
		return ErrEmptyS3Bucket
	}
	if s.S3Region == "" {
		return ErrEmptyS3Region
	}
	if s.S3Key == "" {
		return ErrEmptyS3Key
	}
	if s.S3Secret == "" {
		return ErrEmptyS3Secret
	}

	return nil
}

func NewYandexS3Config() (*YandexS3Config, bool, error) {
	return NewYandexS3ConfigWithLoader(GetConfigLoader())
}

func NewYandexS3ConfigWithLoader(loader ConfigLoader) (*YandexS3Config, bool, error) {
	s3Cfg := &YandexS3Config{loader: loader}
	enabled, err := s3Cfg.LoadConfig()
	return s3Cfg, enabled, err
}
