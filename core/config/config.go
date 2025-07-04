package config

import (
	"os"
)

type (
	Config struct {
		TelegramAccessToken  string
		TelegramWebhookToken string
		TelegramWebhookPort  string
		ServerPanelPassword  string
		ApiUrl               string
		Port                 string
		MongoDB              MongoDB
		Apps                 Apps
		CloudPayments        CloudPayments
	}
	Apps struct {
		AppStoreURL   string
		PlayMarketURL string
		WindowsAppURL string
	}
	MongoDB struct {
		URI      string
		Username string
		Password string
		Name     string
	}
	CloudPayments struct {
		PublicID  string
		SecretKey string
		ApiUrl    string
	}
)

func MustLoad() *Config {
	var cfg Config

	cfg.Apps.AppStoreURL = os.Getenv("APPSTORE_URL")
	cfg.Apps.PlayMarketURL = os.Getenv("PLAYMARKET_URL")
	cfg.Apps.WindowsAppURL = os.Getenv("WINDOWS_APP_URL")
	cfg.TelegramAccessToken = os.Getenv("TELEGRAM_ACCESS_TOKEN")
	cfg.TelegramWebhookToken = os.Getenv("TELEGRAM_WEBHOOK_TOKEN")
	cfg.TelegramWebhookPort = os.Getenv("TELEGRAM_WEBHOOK_PORT")
	cfg.ApiUrl = os.Getenv("API_URL")
	cfg.Port = os.Getenv("PORT")
	cfg.MongoDB.URI = os.Getenv("MONGODB_URI")
	cfg.MongoDB.Username = os.Getenv("MONGODB_USERNAME")
	cfg.MongoDB.Password = os.Getenv("MONGODB_PASSWORD")
	cfg.MongoDB.Name = os.Getenv("MONGODB_NAME")

	cfg.ServerPanelPassword = os.Getenv("SERVER_PANEL_PASSWORD")
	cfg.CloudPayments.PublicID = os.Getenv("CLOUDPAYMENTS_PUBLIC_ID")
	cfg.CloudPayments.SecretKey = os.Getenv("CLOUDPAYMENTS_SECRET")
	cfg.CloudPayments.ApiUrl = os.Getenv("CLOUDPAMENTS_API_URL")

	return &cfg
}
