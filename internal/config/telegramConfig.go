package config

type TelegramConfig struct {
	TelegramEnable bool

	TelegramBotToken string
	TelegramChatID string
}

func (t *TelegramConfig) LoadConfig() (bool, error) {
	t.TelegramEnable = getEnvAsBool("TELEGRAM_ENABLE", false)
	if !t.TelegramEnable {
		return false, nil
	} 

	t.TelegramBotToken = getEnv("TELEGRAM_BOT_TOKEN", "")
	t.TelegramChatID = getEnv("TELEGRAM_CHAT_ID", "")

	err := t.ValidateConfig()

	return true, err
}

// TODO Валидация конфига
func (t *TelegramConfig) ValidateConfig() error {
	return nil;
}

func NewTelegramConfig() (*TelegramConfig, bool, error) {
	telegramCfg := &TelegramConfig{}
	enabled, err := telegramCfg.LoadConfig()
	return telegramCfg, enabled, err
}