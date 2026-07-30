package config

import "errors"

var ErrEmptySQLitePath = errors.New("sqlite path is empty")

type SQLiteConfig struct {
	loader ConfigLoader

	SQLiteEnable bool

	SQLitePath string
}

func (s *SQLiteConfig) LoadConfig() (bool, error) {
	s.SQLiteEnable = s.loader.GetEnvAsBool("SQLITE_ENABLE", false)
	if !s.SQLiteEnable {
		return false, nil
	}

	s.SQLitePath = s.loader.GetEnv("SQLITE_PATH", "./test.db")

	return true, s.ValidateConfig()

}

func (s *SQLiteConfig) ValidateConfig() error {
	if s.SQLitePath == "" {
		return ErrEmptySQLitePath
	}

	return nil
}


func NewSQLiteConfig() (*SQLiteConfig, bool, error) {
	return NewSQLiteConfigWithLoader(&EnvLoader{})
}

func NewSQLiteConfigWithLoader(loader ConfigLoader) (*SQLiteConfig, bool, error) {
	sqliteCfg := &SQLiteConfig{loader: loader}
	enabled, err := sqliteCfg.LoadConfig()
	return sqliteCfg, enabled, err
}
