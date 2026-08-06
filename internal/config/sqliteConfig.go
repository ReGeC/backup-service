package config

import (
	"errors"
)

var ErrEmptySQLitePath = errors.New("sqlite path is empty")

type SQLiteConfig struct {
	loader ConfigLoader

	SQLiteEnable bool

	SQLitePath string
}

func (s *SQLiteConfig) LoadConfig() (bool, error) {
	s.SQLiteEnable = s.loader.GetBool([]string{"sqlite", "enable"}, false)
	if !s.SQLiteEnable {
		return false, nil
	}

	s.SQLitePath = s.loader.GetString([]string{"sqlite", "path"}, "./test.db")

	return true, s.ValidateConfig()

}

func (s *SQLiteConfig) ValidateConfig() error {
	if s.SQLitePath == "" {
		return ErrEmptySQLitePath
	}

	return nil
}

func NewSQLiteConfig() (*SQLiteConfig, bool, error) {
	return NewSQLiteConfigWithLoader(GetConfigLoader())
}

func NewSQLiteConfigWithLoader(loader ConfigLoader) (*SQLiteConfig, bool, error) {
	sqliteCfg := &SQLiteConfig{loader: loader}
	enabled, err := sqliteCfg.LoadConfig()
	return sqliteCfg, enabled, err
}
