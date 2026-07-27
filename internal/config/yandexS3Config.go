package config

type YandexS3Config struct {
	S3Enable bool

	S3Endpoint string
	S3Bucket string
	S3Region string
	S3Key string
	S3Secret string
}

func (s *YandexS3Config) LoadConfig() (bool, error) {
	s.S3Enable = getEnvAsBool("YANDEX_S3_ENABLE", false)
	if !s.S3Enable {
		return false, nil
	}

	s.S3Endpoint = getEnv("YANDEX_S3_ENDPOINT", "https://storage.yandexcloud.net")
	s.S3Bucket = getEnv("YANDEX_S3_BUCKET", "")
	s.S3Region = getEnv("YANDEX_S3_REGION", "ru-central1")
	s.S3Key = getEnv("YANDEX_S3_KEY", "")
	s.S3Secret = getEnv("YANDEX_S3_SECRET", "")

	err := s.ValidateConfig()

	return true, err
}

// TODO Валидация конфига
func (s *YandexS3Config) ValidateConfig() error {
	return nil;
}

func NewYandexS3Config() (*YandexS3Config, bool, error) {
	s3Cfg := &YandexS3Config{}
	enabled, err := s3Cfg.LoadConfig()
	return s3Cfg, enabled, err
}