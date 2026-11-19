package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment          string
	Port                 string
	DatabaseURL          string
	MigrationDatabaseURL string
	AllowedOrigin        string
	FrontendURL          string
	SeedDemo             bool
	WorkerPollInterval   time.Duration
	ReminderPollInterval time.Duration
	Auth                 AuthConfig
	RateLimit            RateLimitConfig
	QuickBooks           QuickBooksConfig
	Email                EmailConfig
}

type AuthConfig struct {
	SessionTTL     time.Duration
	SessionIdleTTL time.Duration
	CookieSecure   bool
}

type RateLimitConfig struct {
	Requests          int
	Window            time.Duration
	LoginRequests     int
	LoginWindow       time.Duration
	SensitiveRequests int
	SensitiveWindow   time.Duration
	TrustedProxyCIDRs []string
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
		Environment:          strings.ToLower(getEnv("APP_ENV", "development")),
		Port:                 getEnv("PORT", "8080"),
		DatabaseURL:          getEnv("DATABASE_URL", "postgres:///ledgerops?host=/var/run/postgresql"),
		MigrationDatabaseURL: os.Getenv("MIGRATION_DATABASE_URL"),
		AllowedOrigin:        getEnv("ALLOWED_ORIGIN", "http://localhost:5173"),
		FrontendURL:          getEnv("FRONTEND_URL", "http://localhost:5173"),
		SeedDemo:             getBool("SEED_DEMO", false),
		WorkerPollInterval:   getDuration("WORKER_POLL_INTERVAL", 2*time.Second),
		ReminderPollInterval: getDuration("REMINDER_POLL_INTERVAL", time.Minute),
		Auth: AuthConfig{
			SessionTTL:     getDuration("AUTH_SESSION_TTL", 12*time.Hour),
			SessionIdleTTL: getDuration("AUTH_SESSION_IDLE_TTL", 30*time.Minute),
			CookieSecure:   getBool("AUTH_COOKIE_SECURE", true),
		},
		RateLimit: RateLimitConfig{
			Requests:          getInt("RATE_LIMIT_REQUESTS", 300),
			Window:            getDuration("RATE_LIMIT_WINDOW", time.Minute),
			LoginRequests:     getInt("LOGIN_RATE_LIMIT_REQUESTS", 10),
			LoginWindow:       getDuration("LOGIN_RATE_LIMIT_WINDOW", 5*time.Minute),
			SensitiveRequests: getInt("SENSITIVE_RATE_LIMIT_REQUESTS", 30),
			SensitiveWindow:   getDuration("SENSITIVE_RATE_LIMIT_WINDOW", time.Minute),
			TrustedProxyCIDRs: splitCSV(os.Getenv("TRUSTED_PROXY_CIDRS")),
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

func (c Config) Validate() error {
	var problems []error
	if c.Environment != "development" && c.Environment != "test" && c.Environment != "production" {
		problems = append(problems, fmt.Errorf("APP_ENV must be development, test, or production"))
	}
	if c.Auth.SessionTTL <= 0 || c.Auth.SessionIdleTTL <= 0 || c.Auth.SessionIdleTTL > c.Auth.SessionTTL {
		problems = append(problems, fmt.Errorf("authentication session durations are invalid"))
	}
	if c.RateLimit.Requests <= 0 || c.RateLimit.Window <= 0 || c.RateLimit.LoginRequests <= 0 || c.RateLimit.LoginWindow <= 0 || c.RateLimit.SensitiveRequests <= 0 || c.RateLimit.SensitiveWindow <= 0 {
		problems = append(problems, fmt.Errorf("rate-limit values must be positive"))
	}
	for name, value := range map[string]string{"ALLOWED_ORIGIN": c.AllowedOrigin, "FRONTEND_URL": c.FrontendURL} {
		parsed, err := url.Parse(value)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || strings.Contains(value, "*") {
			problems = append(problems, fmt.Errorf("%s must be an origin URL without credentials, path, query, or fragment", name))
		}
	}
	quickBooksValues := []string{c.QuickBooks.ClientID, c.QuickBooks.ClientSecret, c.QuickBooks.TokenEncryptionKey}
	configuredQuickBooksValues := 0
	for _, value := range quickBooksValues {
		if value != "" {
			configuredQuickBooksValues++
		}
	}
	if configuredQuickBooksValues != 0 && configuredQuickBooksValues != len(quickBooksValues) {
		problems = append(problems, fmt.Errorf("QuickBooks client ID, secret, and token encryption key must be configured together"))
	}
	if (c.Email.SMTPUsername == "") != (c.Email.SMTPPassword == "") {
		problems = append(problems, fmt.Errorf("SMTP username and password must be configured together"))
	}
	if c.Environment == "production" {
		if c.SeedDemo {
			problems = append(problems, fmt.Errorf("SEED_DEMO must be false in production"))
		}
		if !c.Auth.CookieSecure {
			problems = append(problems, fmt.Errorf("AUTH_COOKIE_SECURE must be true in production"))
		}
		if !c.Email.Configured() {
			problems = append(problems, fmt.Errorf("SMTP_HOST and EMAIL_FROM_ADDRESS are required for password reset in production"))
		}
		for name, value := range map[string]string{"ALLOWED_ORIGIN": c.AllowedOrigin, "FRONTEND_URL": c.FrontendURL} {
			parsed, _ := url.Parse(value)
			if parsed == nil || parsed.Scheme != "https" {
				problems = append(problems, fmt.Errorf("%s must use HTTPS in production", name))
			}
		}
		runtimeDatabaseUser, err := secureDatabaseUser(c.DatabaseURL)
		if err != nil {
			problems = append(problems, fmt.Errorf("DATABASE_URL: %w", err))
		}
		if c.MigrationDatabaseURL == "" || c.MigrationDatabaseURL == c.DatabaseURL {
			problems = append(problems, fmt.Errorf("MIGRATION_DATABASE_URL must use separate migration credentials in production"))
		} else {
			migrationDatabaseUser, migrationErr := secureDatabaseUser(c.MigrationDatabaseURL)
			if migrationErr != nil {
				problems = append(problems, fmt.Errorf("MIGRATION_DATABASE_URL: %w", migrationErr))
			} else if runtimeDatabaseUser != "" && migrationDatabaseUser == runtimeDatabaseUser {
				problems = append(problems, fmt.Errorf("MIGRATION_DATABASE_URL must use a different database role"))
			}
		}
		if c.QuickBooks.Configured() {
			for name, value := range map[string]string{"QUICKBOOKS_REDIRECT_URL": c.QuickBooks.RedirectURL, "QUICKBOOKS_API_BASE_URL": c.QuickBooks.APIBaseURL} {
				parsed, parseErr := url.Parse(value)
				if parseErr != nil || parsed.Scheme != "https" || parsed.Host == "" {
					problems = append(problems, fmt.Errorf("%s must use HTTPS in production", name))
				}
			}
		}
		if c.Email.Configured() && !c.Email.RequireTLS {
			problems = append(problems, fmt.Errorf("SMTP_REQUIRE_TLS must be true when email is enabled in production"))
		}
		if c.Email.Configured() && len(c.Email.WebhookSecret) < 32 {
			problems = append(problems, fmt.Errorf("EMAIL_WEBHOOK_SECRET must contain at least 32 characters when email is enabled in production"))
		}
	}
	return errors.Join(problems...)
}

func secureDatabaseUser(value string) (string, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" || parsed.Host == "" || parsed.User == nil {
		return "", fmt.Errorf("must be a PostgreSQL URL with host and credentials")
	}
	username := parsed.User.Username()
	password, hasPassword := parsed.User.Password()
	if username == "" || !hasPassword || password == "" || strings.EqualFold(username, "ledgerops") && password == "ledgerops" {
		return "", fmt.Errorf("must use non-demo credentials")
	}
	if parsed.Query().Get("sslmode") != "verify-full" {
		return "", fmt.Errorf("must set sslmode=verify-full")
	}
	return username, nil
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
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
