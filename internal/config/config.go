package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port                 int
	DBPath               string
	JWTSecret            string
	Playground           bool
	CORSOrigins          []string
	PagarmeAPIKey        string
	PagarmeWebhookSecret string
	PagarmeRecipientID   string // Platform's own recipient ID for split
	PagarmeAppFee        int64  // centavos per ticket (default 500 = R$5.00)
	BaseURL              string // frontend URL for redirects
}

func Load() *Config {
	port := 8080
	if p := os.Getenv("PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/afterzin.db"
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-in-production"
	}
	playground := os.Getenv("PLAYGROUND") == "true" || os.Getenv("PLAYGROUND") == "1"
	corsOrigins := []string{
		"http://localhost:4040",
		"http://127.0.0.1:4040",
		"http://localhost:5173",
		"http://localhost:3000",
		"http://127.0.0.1:5173",
		"http://127.0.0.1:3000",
		"https://afterzin.com",
		"http://afterzin.com",
	}
	if o := os.Getenv("CORS_ORIGINS"); o != "" {
		// Comma-separated list, e.g. "http://localhost:5173,http://127.0.0.1:5173"
		parts := strings.Split(o, ",")
		corsOrigins = make([]string, 0, len(parts))
		for _, p := range parts {
			if s := strings.TrimSpace(p); s != "" {
				corsOrigins = append(corsOrigins, s)
			}
		}
		if len(corsOrigins) == 0 {
			corsOrigins = []string{"http://localhost:4040", "http://127.0.0.1:4040"}
		}
	}
	stripeSecretKey := os.Getenv("PAGARME_API_KEY")
	stripeWebhookSecret := os.Getenv("PAGARME_WEBHOOK_SECRET")
	pagarmeRecipientID := os.Getenv("PAGARME_PLATFORM_RECIPIENT_ID")
	var stripeAppFee int64 = 500 // R$5.00 default
	if f := os.Getenv("PAGARME_APP_FEE"); f != "" {
		if v, err := strconv.ParseInt(f, 10, 64); err == nil && v > 0 {
			stripeAppFee = v
		}
	}
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:4040"
	}

	return &Config{
		Port:                 port,
		DBPath:               dbPath,
		JWTSecret:            jwtSecret,
		Playground:           playground,
		CORSOrigins:          corsOrigins,
		PagarmeAPIKey:        stripeSecretKey,
		PagarmeWebhookSecret: stripeWebhookSecret,
		PagarmeRecipientID:   pagarmeRecipientID,
		PagarmeAppFee:        stripeAppFee,
		BaseURL:              baseURL,
	}
}
