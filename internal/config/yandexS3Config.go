package config

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
	s.S3Endpoint = s.loader.GetEnv("YANDEX_S3_ENDPOINT", "https://storage.yandexcloud.net")
	s.S3Bucket = s.loader.GetEnv("YANDEX_S3_BUCKET", "")
	s.S3Region = s.loader.GetEnv("YANDEX_S3_REGION", "ru-central1")
	s.S3Key = s.loader.GetEnv("YANDEX_S3_KEY", "")
	s.S3Secret = s.loader.GetEnv("YANDEX_S3_SECRET", "")

	err := s.ValidateConfig()

	return true, err
}

// TODO Валидация конфига
func (s *YandexS3Config) ValidateConfig() error {
	return nil
}

func NewYandexS3Config() (*YandexS3Config, bool, error) {
	return NewYandexS3ConfigWithLoader(&EnvLoader{})
}

func NewYandexS3ConfigWithLoader(loader ConfigLoader) (*YandexS3Config, bool, error) {
	s3Cfg := &YandexS3Config{loader: loader}
	enabled, err := s3Cfg.LoadConfig()
	return s3Cfg, enabled, err
}
