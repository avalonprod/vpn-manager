package config

import (
	"os"
)

type (
	Config struct {
		TelegramAccessToken  string
		TelegramWebhookToken string
		TelegramWebhookPort  string
		ApiUrl               string
		Port                 string
		MongoDB              MongoDB
		Apps                 Apps
	}
	Apps struct {
		AppStoreURL string
	}
	MongoDB struct {
		URI      string
		Username string
		Password string
		Name     string
	}
)

func MustLoad() *Config {
	var cfg Config

	cfg.Apps.AppStoreURL = os.Getenv("APPSTORE_URL")
	cfg.TelegramAccessToken = os.Getenv("TELEGRAM_ACCESS_TOKEN")
	cfg.TelegramWebhookToken = os.Getenv("TELEGRAM_WEBHOOK_TOKEN")
	cfg.TelegramWebhookPort = os.Getenv("TELEGRAM_WEBHOOK_PORT")
	cfg.ApiUrl = os.Getenv("API_URL")
	cfg.Port = os.Getenv("PORT")
	cfg.MongoDB.URI = os.Getenv("MONGODB_URI")
	cfg.MongoDB.Username = os.Getenv("MONGODB_USERNAME")
	cfg.MongoDB.Password = os.Getenv("MONGODB_PASSWORD")
	cfg.MongoDB.Name = os.Getenv("MONGODB_NAME")

	return &cfg
}
