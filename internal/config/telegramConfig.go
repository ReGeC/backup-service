package config

type TelegramConfig struct {
	loader ConfigLoader

	TelegramEnable bool

	TelegramBotToken string
	TelegramChatID   string
}

func (t *TelegramConfig) LoadConfig() (bool, error) {
	t.TelegramEnable = t.loader.GetEnvAsBool("TELEGRAM_ENABLE", false)
	if !t.TelegramEnable {
		return false, nil
	}

	t.TelegramBotToken = t.loader.GetEnv("TELEGRAM_BOT_TOKEN", "")
	t.TelegramChatID = t.loader.GetEnv("TELEGRAM_CHAT_ID", "")

	err := t.ValidateConfig()

	return true, err
}

// TODO Валидация конфига
func (t *TelegramConfig) ValidateConfig() error {
	return nil
}

func NewTelegramConfig() (*TelegramConfig, bool, error) {
	return NewTelegramConfigWithLoader(&EnvLoader{})
}

func NewTelegramConfigWithLoader(loader ConfigLoader) (*TelegramConfig, bool, error) {
	telegramCfg := &TelegramConfig{loader: loader}
	enabled, err := telegramCfg.LoadConfig()
	return telegramCfg, enabled, err
}
