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
		ServerPanelPassword  string
		ApiUrl               string
		Port                 string
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
	// Admin описывает доступ к админ-панели. Учётка одна и живёт в .env.
	Admin struct {
		Enabled bool
		// Username и Password — учётные данные единственного администратора.
		Username string
		Password string
		// PasswordSHA256 — hex-представление sha256(пароль). Позволяет не хранить
		// пароль в открытом виде. Если задано, имеет приоритет над Password.
		PasswordSHA256 string
		// JWTSecret подписывает access-токены (HS256). Минимум 32 байта.
		JWTSecret string
		// TokenTTL — время жизни выданного токена.
		TokenTTL time.Duration
		// AllowedOrigins — белый список Origin для CORS админ-панели.
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

	cfg.ServerPanelPassword = os.Getenv("SERVER_PANEL_PASSWORD")
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

	// Панель включается только при полностью заданных учётных данных, иначе
	// маршруты /admin не регистрируются вовсе.
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
