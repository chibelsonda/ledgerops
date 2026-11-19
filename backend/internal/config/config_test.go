package config

import (
	"strings"
	"testing"
	"time"
)

func validTestConfig() Config {
	return Config{
		Environment: "test", DatabaseURL: "postgres:///ledgerops", AllowedOrigin: "http://localhost:5173",
		FrontendURL: "http://localhost:5173", Auth: AuthConfig{SessionTTL: 12 * time.Hour, SessionIdleTTL: 30 * time.Minute},
		RateLimit:          RateLimitConfig{Requests: 100, Window: time.Minute, LoginRequests: 5, LoginWindow: time.Minute, SensitiveRequests: 10, SensitiveWindow: time.Minute},
		WorkerPollInterval: time.Second, ReminderPollInterval: time.Minute, QuickBooksSchedulePollInterval: time.Minute, QuickBooksOutboxPollInterval: time.Second,
	}
}

func TestProductionConfigRejectsInsecureDefaults(t *testing.T) {
	config := validTestConfig()
	config.Environment = "production"
	config.SeedDemo = true
	config.Auth.CookieSecure = false
	config.DatabaseURL = "postgres://ledgerops:ledgerops@db/ledgerops?sslmode=disable"

	err := config.Validate()
	if err == nil {
		t.Fatal("expected insecure production configuration to fail")
	}
	for _, expected := range []string{"SEED_DEMO", "AUTH_COOKIE_SECURE", "ALLOWED_ORIGIN", "FRONTEND_URL", "DATABASE_URL"} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("validation error %q does not mention %s", err, expected)
		}
	}
}

func TestProductionConfigAcceptsSecureValues(t *testing.T) {
	config := validTestConfig()
	config.Environment = "production"
	config.AllowedOrigin = "https://ledgerops.example"
	config.FrontendURL = "https://ledgerops.example"
	config.DatabaseURL = "postgres://app:strong-password@db/ledgerops?sslmode=verify-full"
	config.MigrationDatabaseURL = "postgres://migrator:different-password@db/ledgerops?sslmode=verify-full"
	config.Auth.CookieSecure = true
	config.Email = EmailConfig{SMTPHost: "smtp.example", SMTPPort: 587, FromAddress: "noreply@ledgerops.example", RequireTLS: true, WebhookSecret: strings.Repeat("a", 32)}

	if err := config.Validate(); err != nil {
		t.Fatalf("validate secure production config: %v", err)
	}
}

func TestConfigRejectsInvalidTrustedOriginShape(t *testing.T) {
	config := validTestConfig()
	config.AllowedOrigin = "https://ledgerops.example/path"
	if err := config.Validate(); err == nil {
		t.Fatal("expected origin containing a path to fail")
	}
}
