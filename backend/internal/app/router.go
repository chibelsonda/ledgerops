package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/portfolio/ledgerops/backend/internal/config"
	"github.com/portfolio/ledgerops/backend/internal/features/adjustments"
	"github.com/portfolio/ledgerops/backend/internal/features/auth"
	"github.com/portfolio/ledgerops/backend/internal/features/clients"
	closefeature "github.com/portfolio/ledgerops/backend/internal/features/close"
	"github.com/portfolio/ledgerops/backend/internal/features/dashboard"
	"github.com/portfolio/ledgerops/backend/internal/features/forecast"
	"github.com/portfolio/ledgerops/backend/internal/features/integrations/quickbooks"
	"github.com/portfolio/ledgerops/backend/internal/features/invoices"
	"github.com/portfolio/ledgerops/backend/internal/features/projects"
	"github.com/portfolio/ledgerops/backend/internal/features/reconciliation"
	"github.com/portfolio/ledgerops/backend/internal/features/reminders"
	"github.com/portfolio/ledgerops/backend/internal/features/sync"
	"github.com/portfolio/ledgerops/backend/internal/platform/httpx"
	"gorm.io/gorm"
)

func NewRouter(db *gorm.DB, logger *slog.Logger, cfg config.Config) (http.Handler, error) {
	mux := http.NewServeMux()
	protectedMux := http.NewServeMux()
	authService, err := auth.NewService(auth.NewPostgresRepository(db), cfg.Auth.SessionTTL, cfg.Auth.SessionIdleTTL)
	if err != nil {
		return nil, fmt.Errorf("configure authentication: %w", err)
	}
	authHandler := auth.NewHandler(authService, logger, cfg.Auth.CookieSecure, cfg.Auth.SessionTTL)
	authMiddleware := auth.NewMiddleware(authService, logger)
	clientIP, err := httpx.NewClientIPResolver(cfg.RateLimit.TrustedProxyCIDRs)
	if err != nil {
		return nil, fmt.Errorf("configure trusted proxies: %w", err)
	}
	globalLimiter := httpx.NewRateLimiter(cfg.RateLimit.Requests, cfg.RateLimit.Window, "global_ip", logger)
	loginLimiter := httpx.NewRateLimiter(cfg.RateLimit.LoginRequests, cfg.RateLimit.LoginWindow, "login_ip", logger)
	sensitiveLimiter := httpx.NewRateLimiter(cfg.RateLimit.SensitiveRequests, cfg.RateLimit.SensitiveWindow, "sensitive_ip", logger)
	userLimiter := httpx.NewRateLimiter(cfg.RateLimit.Requests, cfg.RateLimit.Window, "authenticated_user", logger)
	dashboardRepository := dashboard.NewPostgresRepository(db)
	dashboardHandler := dashboard.NewHandler(dashboard.NewService(dashboardRepository), logger)
	invoicesRepository := invoices.NewPostgresRepository(db)
	invoicesHandler := invoices.NewHandler(invoices.NewService(invoicesRepository), logger)
	adjustmentsRepository := adjustments.NewPostgresRepository(db)
	adjustmentsHandler := adjustments.NewHandler(adjustments.NewService(adjustmentsRepository), logger)
	reconciliationRepository := reconciliation.NewPostgresRepository(db)
	reconciliationHandler := reconciliation.NewHandler(reconciliation.NewService(reconciliationRepository), logger)
	syncRepository := sync.NewPostgresRepository(db)
	syncHandler := sync.NewHandler(sync.NewService(syncRepository), logger)
	clientsRepository := clients.NewPostgresRepository(db)
	clientsHandler := clients.NewHandler(clients.NewService(clientsRepository), logger)
	closeRepository := closefeature.NewPostgresRepository(db)
	closeHandler := closefeature.NewHandler(closefeature.NewService(closeRepository), logger)
	forecastRepository := forecast.NewPostgresRepository(db)
	forecastHandler := forecast.NewHandler(forecast.NewService(forecastRepository), logger)
	projectsRepository := projects.NewPostgresRepository(db)
	projectsHandler := projects.NewHandler(projects.NewService(projectsRepository), logger)
	remindersRepository := reminders.NewPostgresRepository(db)
	remindersHandler := reminders.NewHandler(
		reminders.NewService(remindersRepository),
		logger,
		reminders.DeliveryStatus{Configured: cfg.Email.Configured(), FromAddress: cfg.Email.FromAddress},
		cfg.Email.WebhookSecret,
	)
	quickBooksRepository := quickbooks.NewPostgresRepository(db)
	var quickBooksClient quickbooks.Client
	var tokenCipher quickbooks.Cipher
	quickBooksConfigured := cfg.QuickBooks.Configured()
	if quickBooksConfigured {
		cipher, err := quickbooks.NewAESGCMCipher(cfg.QuickBooks.TokenEncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("configure QuickBooks integration: %w", err)
		}
		tokenCipher = cipher
		quickBooksClient = quickbooks.NewIntuitClient(
			cfg.QuickBooks.ClientID,
			cfg.QuickBooks.ClientSecret,
			cfg.QuickBooks.RedirectURL,
			cfg.QuickBooks.APIBaseURL,
		)
	}
	quickBooksService := quickbooks.NewService(quickBooksRepository, quickBooksClient, tokenCipher, quickBooksConfigured)
	quickBooksHandler := quickbooks.NewHandler(quickBooksService, logger, cfg.FrontendURL)

	mux.HandleFunc("GET /healthz", httpx.HealthHandler(func(ctx context.Context) error {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return sqlDB.PingContext(ctx)
	}, logger))
	mux.Handle("POST /api/v1/auth/login", loginLimiter.Middleware(http.HandlerFunc(authHandler.Login), clientIP.Key))
	mux.HandleFunc("GET /api/v1/auth/session", authHandler.Session)
	mux.Handle("GET /api/v1/integrations/quickbooks/callback", sensitiveLimiter.Middleware(http.HandlerFunc(quickBooksHandler.Callback), clientIP.Key))
	mux.Handle("POST /api/v1/integrations/email/webhook", sensitiveLimiter.Middleware(http.HandlerFunc(remindersHandler.DeliveryWebhook), clientIP.Key))

	registerProtectedRoutes(protectedMux, logger, routeHandlers{
		auth: authHandler, dashboard: dashboardHandler, invoices: invoicesHandler, adjustments: adjustmentsHandler,
		clients: clientsHandler, projects: projectsHandler, close: closeHandler, forecast: forecastHandler,
		reminders: remindersHandler, reconciliation: reconciliationHandler, sync: syncHandler, quickBooks: quickBooksHandler,
	})
	mux.Handle("/api/v1/", authMiddleware.Authenticate(userLimiter.Middleware(protectedMux, httpx.PrincipalRateLimitKey)))

	return httpx.Wrap(globalLimiter.Middleware(mux, clientIP.Key), logger, cfg.AllowedOrigin), nil
}
