package config

type PostgresConfig struct {
	PGEnable bool

	PGHost string
	PGPort int
	PGUser string
	PGPassword string
	PGDatabase string
}

func (p *PostgresConfig) LoadConfig() (bool, error) {
	p.PGEnable = getEnvAsBool("PG_ENABLE", false)
	if !p.PGEnable {
		return false, nil
	} 

	p.PGHost = getEnv("PG_HOST", "localhost")
	p.PGPort = getEnvAsInt("PG_PORT", 5432)
	p.PGUser = getEnv("PG_USER", "postgres")
	p.PGPassword = getEnv("PG_PASSWORD", "")
	p.PGDatabase = getEnv("PG_DATABASE", "postgres")

	err := p.ValidateConfig()

	return true, err
}

// TODO Валидация конфига
func (p *PostgresConfig) ValidateConfig() error {
	return nil;
}

func NewPostgresConfig() (*PostgresConfig, bool, error) {
	postgresCfg := &PostgresConfig{}
	enabled, err := postgresCfg.LoadConfig()
	return postgresCfg, enabled, err
}