package config

type S3Config struct {
	S3Bucket string
	S3Region string
	S3Key string
	S3Secret string
}

func (s *S3Config) LoadConfig() error {
	s.S3Bucket = getEnv("S3_BUCKET", "")
	s.S3Region = getEnv("S3_REGION", "")
	s.S3Key = getEnv("S3_KEY", "")
	s.S3Secret = getEnv("S3_SECRET", "")

	err := s.ValidateConfig()

	return err
}

// TODO Валидация конфига
func (s *S3Config) ValidateConfig() error {
	return nil;
}

func NewS3Config() (*S3Config, error) {
	s3Cfg := &S3Config{}
	err := s3Cfg.LoadConfig()
	return s3Cfg, err
}