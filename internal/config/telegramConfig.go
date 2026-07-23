package config

type TelegramConfig struct {
	TelegramBotToken string
	TelegramChatID string
}

func (t *TelegramConfig) LoadConfig() error {
	t.TelegramBotToken = getEnv("TELEGRAM_BOT_TOKEN", "")
	t.TelegramChatID = getEnv("TELEGRAM_CHAT_ID", "")

	err := t.ValidateConfig()

	return err
}

// TODO Валидация конфига
func (t *TelegramConfig) ValidateConfig() error {
	return nil;
}

func NewTelegramConfig() (*TelegramConfig, error) {
	telegramCfg := &TelegramConfig{}
	err := telegramCfg.LoadConfig()
	return telegramCfg, err
}