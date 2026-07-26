package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type (
	Config struct {
		TelegramAccessToken  string
		TelegramWebhookToken string
		TelegramWebhookPort  string
		ApiUrl               string
		Port                 string
		ServerTokenEncKey    string
		AllowLegacySubLinks  bool
		MongoDB              MongoDB
		Apps                 Apps
		CloudPayments        CloudPayments
		Admin                Admin
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

	Admin struct {
		Enabled bool

		Username string
		Password string

		PasswordSHA256 string

		JWTSecret string

		TokenTTL time.Duration

		AllowedOrigins []string
	}
)

func MustLoad() *Config {
	godotenv.Load()
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

	cfg.ServerTokenEncKey = strings.TrimSpace(os.Getenv("SERVER_TOKEN_ENC_KEY"))
	if cfg.ServerTokenEncKey == "" {
		log.Fatal("SERVER_TOKEN_ENC_KEY is required: generate one with `openssl rand -hex 32`")
	}

	cfg.AllowLegacySubLinks = os.Getenv("ALLOW_LEGACY_SUB_LINKS") == "true"
	if cfg.AllowLegacySubLinks {
		log.Print("WARNING: ALLOW_LEGACY_SUB_LINKS=true — /subs and /setup still accept an unauthenticated user_id, anyone can fetch another user's configs")
	}

	cfg.CloudPayments.PublicID = os.Getenv("CLOUDPAYMENTS_PUBLIC_ID")
	cfg.CloudPayments.SecretKey = os.Getenv("CLOUDPAYMENTS_SECRET")
	cfg.CloudPayments.ApiUrl = os.Getenv("CLOUDPAMENTS_API_URL")

	cfg.Admin = loadAdmin()

	return &cfg
}

const (
	defaultAdminTokenTTL = 8 * time.Hour
	minJWTSecretLen      = 32
)

func loadAdmin() Admin {
	admin := Admin{
		Username:       strings.TrimSpace(os.Getenv("ADMIN_USERNAME")),
		Password:       os.Getenv("ADMIN_PASSWORD"),
		PasswordSHA256: strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_PASSWORD_SHA256"))),
		JWTSecret:      os.Getenv("ADMIN_JWT_SECRET"),
		TokenTTL:       defaultAdminTokenTTL,
	}

	for _, origin := range strings.Split(os.Getenv("ADMIN_CORS_ORIGINS"), ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			admin.AllowedOrigins = append(admin.AllowedOrigins, strings.TrimRight(origin, "/"))
		}
	}

	if raw := os.Getenv("ADMIN_TOKEN_TTL_MINUTES"); raw != "" {
		minutes, err := strconv.Atoi(raw)
		if err != nil || minutes <= 0 {
			log.Fatal("Invalid ADMIN_TOKEN_TTL_MINUTES: must be a positive integer")
		}
		admin.TokenTTL = time.Duration(minutes) * time.Minute
	}

	if admin.Username == "" || (admin.Password == "" && admin.PasswordSHA256 == "") {
		log.Print("admin panel is disabled: ADMIN_USERNAME and ADMIN_PASSWORD (or ADMIN_PASSWORD_SHA256) are not set")
		return Admin{}
	}

	if len(admin.JWTSecret) < minJWTSecretLen {
		log.Fatalf("Invalid ADMIN_JWT_SECRET: at least %d characters required", minJWTSecretLen)
	}

	admin.Enabled = true

	return admin
}
