package config

type SQLiteConfig struct {
	SQLitePath string
}

func (s *SQLiteConfig) LoadConfig() error {
	s.SQLitePath = getEnv("SQLITE_PATH", "./test.db")

	err := s.ValidateConfig()

	return err
}

// TODO Валидация конфига
func (s *SQLiteConfig) ValidateConfig() error {
	return nil;
}

func NewSQLiteConfig() (*SQLiteConfig, error) {
	sqliteCfg := &SQLiteConfig{}
	err := sqliteCfg.LoadConfig()
	return sqliteCfg, err
}