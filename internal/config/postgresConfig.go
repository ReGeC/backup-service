package config

import (
	"errors"
)

var (
	ErrEmptyPGHost     = errors.New("postgres host is empty")
	ErrInvalidPGPort   = errors.New("postgres port must be between 1 and 65535")
	ErrEmptyPGUser     = errors.New("postgres user is empty")
	ErrEmptyPGDatabase = errors.New("postgres database is empty")
)

type PostgresConfig struct {
	loader ConfigLoader

	PGEnable bool

	PGHost     string
	PGPort     int
	PGUser     string
	PGPassword string
	PGDatabase string
}

func (p *PostgresConfig) LoadConfig() (bool, error) {
	p.PGEnable = p.loader.GetBool([]string{"postgres", "enable"}, false)
	if !p.PGEnable {
		return false, nil
	}

	p.PGHost = p.loader.GetString([]string{"postgres", "host"}, "localhost")
	p.PGPort = p.loader.GetInt([]string{"postgres", "port"}, 5432)
	p.PGUser = p.loader.GetString([]string{"postgres", "user"}, "postgres")
	p.PGPassword = p.loader.GetString([]string{"postgres", "password"}, "")
	p.PGDatabase = p.loader.GetString([]string{"postgres", "database"}, "postgres")

	return true, p.ValidateConfig()
}

func (p *PostgresConfig) ValidateConfig() error {
	if p.PGHost == "" {
		return ErrEmptyPGHost
	}

	if p.PGPort < 1 || p.PGPort > 65535 {
		return ErrInvalidPGPort
	}

	if p.PGUser == "" {
		return ErrEmptyPGUser
	}

	if p.PGDatabase == "" {
		return ErrEmptyPGDatabase
	}

	return nil
}

func NewPostgresConfig() (*PostgresConfig, bool, error) {
	return NewPostgresConfigWithLoader(GetConfigLoader())
}

func NewPostgresConfigWithLoader(loader ConfigLoader) (*PostgresConfig, bool, error) {
	postgresCfg := &PostgresConfig{loader: loader}
	enabled, err := postgresCfg.LoadConfig()
	return postgresCfg, enabled, err
}
