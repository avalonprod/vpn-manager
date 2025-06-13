package config

import (
	"os"
)

type (
	Config struct {
		TelegramAccessToken string
		ApiUrl              string
		Port                string
		MongoDB             MongoDB
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

	cfg.TelegramAccessToken = os.Getenv("TELEGRAM_ACCESS_TOKEN")
	cfg.ApiUrl = os.Getenv("API_URL")
	cfg.Port = os.Getenv("PORT")
	cfg.MongoDB.URI = os.Getenv("MONGODB_URI")
	cfg.MongoDB.Username = os.Getenv("MONGODB_USERNAME")
	cfg.MongoDB.Password = os.Getenv("MONGODB_PASSWORD")
	cfg.MongoDB.Name = os.Getenv("MONGODB_NAME")

	return &cfg
}
