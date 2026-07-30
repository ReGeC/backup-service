package config

import "errors"

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
	p.PGEnable = p.loader.GetEnvAsBool("PG_ENABLE", false)
	if !p.PGEnable {
		return false, nil
	}

	p.PGHost = p.loader.GetEnv("PG_HOST", "localhost")
	p.PGPort = p.loader.GetEnvAsInt("PG_PORT", 5432)
	p.PGUser = p.loader.GetEnv("PG_USER", "postgres")
	p.PGPassword = p.loader.GetEnv("PG_PASSWORD", "")
	p.PGDatabase = p.loader.GetEnv("PG_DATABASE", "postgres")

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
	return NewPostgresConfigWithLoader(&EnvLoader{})
}

func NewPostgresConfigWithLoader(loader ConfigLoader) (*PostgresConfig, bool, error) {
	postgresCfg := &PostgresConfig{loader: loader}
	enabled, err := postgresCfg.LoadConfig()
	return postgresCfg, enabled, err
}
