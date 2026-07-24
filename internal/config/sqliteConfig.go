package config

type SQLiteConfig struct {
	SQLiteEnable bool

	SQLitePath string
}

func (s *SQLiteConfig) LoadConfig() (bool, error) {
	s.SQLiteEnable = getEnvAsBool("SQLITE_ENABLE", false)
	if !s.SQLiteEnable {
		return false, nil
	} 

	s.SQLitePath = getEnv("SQLITE_PATH", "./test.db")
	err := s.ValidateConfig()

	return true, err
}

// TODO Валидация конфига
func (s *SQLiteConfig) ValidateConfig() error {
	return nil;
}

func NewSQLiteConfig() (*SQLiteConfig, bool, error) {
	sqliteCfg := &SQLiteConfig{}
	enabled, err := sqliteCfg.LoadConfig()
	return sqliteCfg, enabled, err
}