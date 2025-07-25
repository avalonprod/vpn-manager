package config

import (
	"log"
	"os"
	"strconv"
)

type (
	Config struct {
		TelegramAccessToken    string
		TelegramWebhookToken   string
		TelegramWebhookPort    string
		ServerPanelPassword    string
		ApiUrl                 string
		Port                   string
		AnalyticsChanelID      int
		AnalyticsSpreadsheetID string
		GoogleCredentials      string
		MongoDB                MongoDB
		Apps                   Apps
		CloudPayments          CloudPayments
	}
	Apps struct {
		AppStoreURL   string
		PlayMarketURL string
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
	cfg.TelegramAccessToken = os.Getenv("TELEGRAM_ACCESS_TOKEN")
	cfg.TelegramWebhookToken = os.Getenv("TELEGRAM_WEBHOOK_TOKEN")
	cfg.TelegramWebhookPort = os.Getenv("TELEGRAM_WEBHOOK_PORT")
	cfg.ApiUrl = os.Getenv("API_URL")
	cfg.Port = os.Getenv("PORT")
	cfg.MongoDB.URI = os.Getenv("MONGODB_URI")
	cfg.MongoDB.Username = os.Getenv("MONGODB_USERNAME")
	cfg.MongoDB.Password = os.Getenv("MONGODB_PASSWORD")
	cfg.MongoDB.Name = os.Getenv("MONGODB_NAME")

	cfg.GoogleCredentials = os.Getenv("GOOGLE_CREDENTIALS")

	cfg.ServerPanelPassword = os.Getenv("SERVER_PANEL_PASSWORD")
	cfg.CloudPayments.PublicID = os.Getenv("CLOUDPAYMENTS_PUBLIC_ID")
	cfg.CloudPayments.SecretKey = os.Getenv("CLOUDPAYMENTS_SECRET")
	cfg.CloudPayments.ApiUrl = os.Getenv("CLOUDPAMENTS_API_URL")

	analyticsChanelID, err := strconv.Atoi(os.Getenv("ANALYTICS_CHANNEL_ID"))
	if err != nil {
		log.Fatal("Invalid ANALYTICS_CHANNEL_ID:", err)
	}
	cfg.AnalyticsChanelID = analyticsChanelID
	cfg.AnalyticsSpreadsheetID = os.Getenv("ANALYTICS_SPREADSHEET_ID")

	return &cfg
}
