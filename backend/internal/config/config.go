package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                 string
	DatabaseURL          string
	AllowedOrigin        string
	FrontendURL          string
	SeedDemo             bool
	WorkerPollInterval   time.Duration
	ReminderPollInterval time.Duration
	Auth                 AuthConfig
	QuickBooks           QuickBooksConfig
	Email                EmailConfig
}

type AuthConfig struct {
	SessionTTL   time.Duration
	CookieSecure bool
}

type QuickBooksConfig struct {
	ClientID           string
	ClientSecret       string
	RedirectURL        string
	APIBaseURL         string
	TokenEncryptionKey string
}

type EmailConfig struct {
	SMTPHost             string
	SMTPPort             int
	SMTPUsername         string
	SMTPPassword         string
	FromAddress          string
	FromName             string
	RequireTLS           bool
	WebhookSecret        string
	DeliveryPollInterval time.Duration
}

func (c EmailConfig) Configured() bool {
	return c.SMTPHost != "" && c.SMTPPort > 0 && c.FromAddress != ""
}

func (c QuickBooksConfig) Configured() bool {
	return c.ClientID != "" && c.ClientSecret != "" && c.TokenEncryptionKey != ""
}

func Load() Config {
	return Config{
		Port:                 getEnv("PORT", "8080"),
		DatabaseURL:          getEnv("DATABASE_URL", "postgres:///ledgerops?host=/var/run/postgresql"),
		AllowedOrigin:        getEnv("ALLOWED_ORIGIN", "http://localhost:5173"),
		FrontendURL:          getEnv("FRONTEND_URL", "http://localhost:5173"),
		SeedDemo:             getBool("SEED_DEMO", true),
		WorkerPollInterval:   getDuration("WORKER_POLL_INTERVAL", 2*time.Second),
		ReminderPollInterval: getDuration("REMINDER_POLL_INTERVAL", time.Minute),
		Auth: AuthConfig{
			SessionTTL:   getDuration("AUTH_SESSION_TTL", 12*time.Hour),
			CookieSecure: getBool("AUTH_COOKIE_SECURE", false),
		},
		QuickBooks: QuickBooksConfig{
			ClientID:           os.Getenv("QUICKBOOKS_CLIENT_ID"),
			ClientSecret:       os.Getenv("QUICKBOOKS_CLIENT_SECRET"),
			RedirectURL:        getEnv("QUICKBOOKS_REDIRECT_URL", "http://localhost:8080/api/v1/integrations/quickbooks/callback"),
			APIBaseURL:         getEnv("QUICKBOOKS_API_BASE_URL", "https://sandbox-quickbooks.api.intuit.com"),
			TokenEncryptionKey: os.Getenv("QUICKBOOKS_TOKEN_ENCRYPTION_KEY"),
		},
		Email: EmailConfig{
			SMTPHost:             os.Getenv("SMTP_HOST"),
			SMTPPort:             getInt("SMTP_PORT", 587),
			SMTPUsername:         os.Getenv("SMTP_USERNAME"),
			SMTPPassword:         os.Getenv("SMTP_PASSWORD"),
			FromAddress:          os.Getenv("EMAIL_FROM_ADDRESS"),
			FromName:             getEnv("EMAIL_FROM_NAME", "LedgerOps"),
			RequireTLS:           getBool("SMTP_REQUIRE_TLS", true),
			WebhookSecret:        os.Getenv("EMAIL_WEBHOOK_SECRET"),
			DeliveryPollInterval: getDuration("EMAIL_DELIVERY_POLL_INTERVAL", 5*time.Second),
		},
	}
}

func getInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
