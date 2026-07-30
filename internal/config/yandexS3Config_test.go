package config_test

import (
	"testing"

	"backup-service/internal/config"
	mocks "backup-service/internal/config/mocks"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
)

func TestYandexS3Config_ValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  config.YandexS3Config
		wantErr error
	}{
		{
			name: "valid config",
			config: config.YandexS3Config{
				S3Endpoint: "https://storage.yandexcloud.net",
				S3Bucket:   "my-backup-bucket",
				S3Region:   "ru-central1",
				S3Key:      "access-key",
				S3Secret:   "secret-key",
			},
			wantErr: nil,
		},
		{
			name: "empty endpoint",
			config: config.YandexS3Config{
				S3Endpoint: "",
				S3Bucket:   "my-backup-bucket",
				S3Region:   "ru-central1",
				S3Key:      "access-key",
				S3Secret:   "secret-key",
			},
			wantErr: config.ErrEmptyS3Endpoint,
		},
		{
			name: "invalid endpoint - no http prefix",
			config: config.YandexS3Config{
				S3Endpoint: "storage.yandexcloud.net",
				S3Bucket:   "my-backup-bucket",
				S3Region:   "ru-central1",
				S3Key:      "access-key",
				S3Secret:   "secret-key",
			},
			wantErr: config.ErrInvalidS3Endpoint,
		},
		{
			name: "invalid endpoint - ftp prefix",
			config: config.YandexS3Config{
				S3Endpoint: "ftp://storage.yandexcloud.net",
				S3Bucket:   "my-backup-bucket",
				S3Region:   "ru-central1",
				S3Key:      "access-key",
				S3Secret:   "secret-key",
			},
			wantErr: config.ErrInvalidS3Endpoint,
		},
		{
			name: "empty bucket",
			config: config.YandexS3Config{
				S3Endpoint: "https://storage.yandexcloud.net",
				S3Bucket:   "",
				S3Region:   "ru-central1",
				S3Key:      "access-key",
				S3Secret:   "secret-key",
			},
			wantErr: config.ErrEmptyS3Bucket,
		},
		{
			name: "empty region",
			config: config.YandexS3Config{
				S3Endpoint: "https://storage.yandexcloud.net",
				S3Bucket:   "my-backup-bucket",
				S3Region:   "",
				S3Key:      "access-key",
				S3Secret:   "secret-key",
			},
			wantErr: config.ErrEmptyS3Region,
		},
		{
			name: "empty key",
			config: config.YandexS3Config{
				S3Endpoint: "https://storage.yandexcloud.net",
				S3Bucket:   "my-backup-bucket",
				S3Region:   "ru-central1",
				S3Key:      "",
				S3Secret:   "secret-key",
			},
			wantErr: config.ErrEmptyS3Key,
		},
		{
			name: "empty secret",
			config: config.YandexS3Config{
				S3Endpoint: "https://storage.yandexcloud.net",
				S3Bucket:   "my-backup-bucket",
				S3Region:   "ru-central1",
				S3Key:      "access-key",
				S3Secret:   "",
			},
			wantErr: config.ErrEmptyS3Secret,
		},
		{
			name: "valid endpoint with http",
			config: config.YandexS3Config{
				S3Endpoint: "http://storage.yandexcloud.net",
				S3Bucket:   "my-backup-bucket",
				S3Region:   "ru-central1",
				S3Key:      "access-key",
				S3Secret:   "secret-key",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateConfig()

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestNewYandexS3ConfigWithLoader(t *testing.T) {
	tests := []struct {
		name          string
		s3Endpoint    string
		s3Bucket      string
		s3Region      string
		s3Key         string
		s3Secret      string
		wantEnabled   bool
	}{
		{
			name:        "full config with default endpoint",
			s3Endpoint:  "https://storage.yandexcloud.net",
			s3Bucket:    "my-backup-bucket",
			s3Region:    "ru-central1",
			s3Key:       "access-key",
			s3Secret:    "secret-key",
			wantEnabled: true,
		},
		{
			name:        "full config with custom endpoint",
			s3Endpoint:  "https://custom-storage.example.com",
			s3Bucket:    "my-backup-bucket",
			s3Region:    "ru-central1",
			s3Key:       "access-key",
			s3Secret:    "secret-key",
			wantEnabled: true,
		},
		{
			name:        "empty bucket should still return enabled true but error on validate",
			s3Endpoint:  "https://storage.yandexcloud.net",
			s3Bucket:    "",
			s3Region:    "ru-central1",
			s3Key:       "access-key",
			s3Secret:    "secret-key",
			wantEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := mocks.NewMockConfigLoader(t)

			// YandexS3Config всегда загружает все переменные, независимо от enabled
			loader.EXPECT().
				GetEnv("YANDEX_S3_ENDPOINT", "https://storage.yandexcloud.net").
				Return(tt.s3Endpoint)

			loader.EXPECT().
				GetEnv("YANDEX_S3_BUCKET", "").
				Return(tt.s3Bucket)

			loader.EXPECT().
				GetEnv("YANDEX_S3_REGION", "ru-central1").
				Return(tt.s3Region)

			loader.EXPECT().
				GetEnv("YANDEX_S3_KEY", "").
				Return(tt.s3Key)

			loader.EXPECT().
				GetEnv("YANDEX_S3_SECRET", "").
				Return(tt.s3Secret)

			cfg, enabled, err := config.NewYandexS3ConfigWithLoader(loader)

			require.NotNil(t, cfg)
			assert.Equal(t, tt.wantEnabled, enabled)

			// Проверяем, что ошибка может быть только от ValidateConfig
			if tt.s3Bucket == "" {
				require.ErrorIs(t, err, config.ErrEmptyS3Bucket)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.s3Endpoint, cfg.S3Endpoint)
			assert.Equal(t, tt.s3Bucket, cfg.S3Bucket)
			assert.Equal(t, tt.s3Region, cfg.S3Region)
			assert.Equal(t, tt.s3Key, cfg.S3Key)
			assert.Equal(t, tt.s3Secret, cfg.S3Secret)
		})
	}
}

func TestNewYandexS3Config(t *testing.T) {
	t.Run("Создание yandex s3 конфига с реальным envloader", func(t *testing.T) {
		// Устанавливаем переменные окружения для теста
		t.Setenv("YANDEX_S3_ENDPOINT", "https://storage.yandexcloud.net")
		t.Setenv("YANDEX_S3_BUCKET", "my-backup-bucket")
		t.Setenv("YANDEX_S3_REGION", "ru-central1")
		t.Setenv("YANDEX_S3_KEY", "test-access-key")
		t.Setenv("YANDEX_S3_SECRET", "test-secret-key")

		cfg, enabled, err := config.NewYandexS3Config()

		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.True(t, enabled) // Всегда true, так как нет Enable

		// Проверяем, что значения загрузились корректно
		assert.Equal(t, "https://storage.yandexcloud.net", cfg.S3Endpoint)
		assert.Equal(t, "my-backup-bucket", cfg.S3Bucket)
		assert.Equal(t, "ru-central1", cfg.S3Region)
		assert.Equal(t, "test-access-key", cfg.S3Key)
		assert.Equal(t, "test-secret-key", cfg.S3Secret)
	})
}
