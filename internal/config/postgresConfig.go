package config

type PostgresConfig struct {
	PGHost string
	PGPort int
	PGUser string
	PGPassword string
	PGDatabase string
}

func (p *PostgresConfig) LoadConfig() error {
	p.PGHost = getEnv("PG_HOST", "localhost")
	p.PGPort = getEnvAsInt("PG_PORT", 5432)
	p.PGUser = getEnv("PG_USER", "postgres")
	p.PGPassword = getEnv("PG_PASSWORD", "")
	p.PGDatabase = getEnv("PG_DATABASE", "postgres")

	err := p.ValidateConfig()

	return err
}

// TODO Валидация конфига
func (p *PostgresConfig) ValidateConfig() error {
	return nil;
}

func NewPostgresConfig() (*PostgresConfig, error) {
	postgresCfg := &PostgresConfig{}
	err := postgresCfg.LoadConfig()
	return postgresCfg, err
}