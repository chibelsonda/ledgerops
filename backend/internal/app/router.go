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
	"github.com/portfolio/ledgerops/backend/internal/features/dashboard"
	"github.com/portfolio/ledgerops/backend/internal/features/integrations/quickbooks"
	"github.com/portfolio/ledgerops/backend/internal/features/invoices"
	"github.com/portfolio/ledgerops/backend/internal/features/projects"
	"github.com/portfolio/ledgerops/backend/internal/features/reconciliation"
	"github.com/portfolio/ledgerops/backend/internal/features/reminders"
	"github.com/portfolio/ledgerops/backend/internal/features/sync"
	"github.com/portfolio/ledgerops/backend/internal/platform/httpx"
	"github.com/portfolio/ledgerops/backend/internal/platform/identity"
	"gorm.io/gorm"
)

func NewRouter(db *gorm.DB, logger *slog.Logger, cfg config.Config) (http.Handler, error) {
	mux := http.NewServeMux()
	protectedMux := http.NewServeMux()
	authService, err := auth.NewService(auth.NewPostgresRepository(db), cfg.Auth.SessionTTL)
	if err != nil {
		return nil, fmt.Errorf("configure authentication: %w", err)
	}
	authHandler := auth.NewHandler(authService, logger, cfg.Auth.CookieSecure, cfg.Auth.SessionTTL)
	authMiddleware := auth.NewMiddleware(authService, logger)
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
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("GET /api/v1/auth/session", authHandler.Session)
	mux.HandleFunc("GET /api/v1/integrations/quickbooks/callback", quickBooksHandler.Callback)
	mux.HandleFunc("POST /api/v1/integrations/email/webhook", remindersHandler.DeliveryWebhook)

	allRoles := auth.RequireRoles(identity.RoleAdmin, identity.RoleController, identity.RoleReviewer)
	operators := auth.RequireRoles(identity.RoleAdmin, identity.RoleController)
	admins := auth.RequireRoles(identity.RoleAdmin)
	protectedMux.Handle("GET /api/v1/dashboard", allRoles(http.HandlerFunc(dashboardHandler.Get)))
	protectedMux.Handle("GET /api/v1/invoices", allRoles(http.HandlerFunc(invoicesHandler.List)))
	protectedMux.Handle("GET /api/v1/invoices/clients", allRoles(http.HandlerFunc(invoicesHandler.ListClients)))
	protectedMux.Handle("POST /api/v1/invoices", operators(http.HandlerFunc(invoicesHandler.Create)))
	protectedMux.Handle("PATCH /api/v1/invoices/{invoiceID}", operators(http.HandlerFunc(invoicesHandler.Update)))
	protectedMux.Handle("GET /api/v1/invoices/{invoiceID}/audit-events", allRoles(http.HandlerFunc(invoicesHandler.ListAuditEvents)))
	protectedMux.Handle("GET /api/v1/invoices/{invoiceID}/payments", allRoles(http.HandlerFunc(invoicesHandler.ListPayments)))
	protectedMux.Handle("POST /api/v1/invoices/{invoiceID}/payments", operators(http.HandlerFunc(invoicesHandler.RecordPayment)))
	protectedMux.Handle("GET /api/v1/invoices/{invoiceID}/adjustments", allRoles(http.HandlerFunc(adjustmentsHandler.List)))
	protectedMux.Handle("POST /api/v1/invoices/{invoiceID}/adjustments", operators(http.HandlerFunc(adjustmentsHandler.Create)))
	protectedMux.Handle("PATCH /api/v1/adjustments/{adjustmentID}", admins(http.HandlerFunc(adjustmentsHandler.Decide)))
	protectedMux.Handle("GET /api/v1/clients", allRoles(http.HandlerFunc(clientsHandler.List)))
	protectedMux.Handle("POST /api/v1/clients", operators(http.HandlerFunc(clientsHandler.Create)))
	protectedMux.Handle("PATCH /api/v1/clients/{clientID}", operators(http.HandlerFunc(clientsHandler.Update)))
	protectedMux.Handle("GET /api/v1/clients/{clientID}/contacts", allRoles(http.HandlerFunc(clientsHandler.ListContacts)))
	protectedMux.Handle("POST /api/v1/clients/{clientID}/contacts", operators(http.HandlerFunc(clientsHandler.CreateContact)))
	protectedMux.Handle("PATCH /api/v1/contacts/{contactID}", operators(http.HandlerFunc(clientsHandler.UpdateContact)))
	protectedMux.Handle("GET /api/v1/projects", allRoles(http.HandlerFunc(projectsHandler.List)))
	protectedMux.Handle("GET /api/v1/projects/clients", allRoles(http.HandlerFunc(projectsHandler.ListClients)))
	protectedMux.Handle("GET /api/v1/projects/{projectID}", allRoles(http.HandlerFunc(projectsHandler.Get)))
	protectedMux.Handle("POST /api/v1/projects", operators(http.HandlerFunc(projectsHandler.Create)))
	protectedMux.Handle("PATCH /api/v1/contracts/{contractID}/decision", admins(http.HandlerFunc(projectsHandler.DecideContract)))
	protectedMux.Handle("POST /api/v1/contracts/{contractID}/change-orders", operators(http.HandlerFunc(projectsHandler.CreateChangeOrder)))
	protectedMux.Handle("PATCH /api/v1/change-orders/{changeOrderID}/decision", admins(http.HandlerFunc(projectsHandler.DecideChangeOrder)))
	protectedMux.Handle("POST /api/v1/contracts/{contractID}/milestones", operators(http.HandlerFunc(projectsHandler.CreateMilestone)))
	protectedMux.Handle("POST /api/v1/milestones/{milestoneID}/invoice", operators(http.HandlerFunc(projectsHandler.InvoiceMilestone)))
	protectedMux.Handle("GET /api/v1/reminders", allRoles(http.HandlerFunc(remindersHandler.List)))
	protectedMux.Handle("GET /api/v1/reminders/delivery-status", allRoles(http.HandlerFunc(remindersHandler.DeliveryStatus)))
	protectedMux.Handle("POST /api/v1/reminders/generate", operators(http.HandlerFunc(remindersHandler.Generate)))
	protectedMux.Handle("PATCH /api/v1/reminders/{reminderID}/cancel", operators(http.HandlerFunc(remindersHandler.Cancel)))
	protectedMux.Handle("GET /api/v1/reconciliation", allRoles(http.HandlerFunc(reconciliationHandler.List)))
	protectedMux.Handle("PATCH /api/v1/reconciliation/{itemID}", allRoles(http.HandlerFunc(reconciliationHandler.Resolve)))
	protectedMux.Handle("GET /api/v1/sync-runs", operators(http.HandlerFunc(syncHandler.List)))
	protectedMux.Handle("POST /api/v1/sync-runs", operators(http.HandlerFunc(syncHandler.Enqueue)))
	protectedMux.Handle("GET /api/v1/integrations/quickbooks", operators(http.HandlerFunc(quickBooksHandler.Status)))
	protectedMux.Handle("POST /api/v1/integrations/quickbooks/connect", admins(http.HandlerFunc(quickBooksHandler.Connect)))
	protectedMux.Handle("DELETE /api/v1/integrations/quickbooks", admins(http.HandlerFunc(quickBooksHandler.Disconnect)))
	protectedMux.HandleFunc("POST /api/v1/auth/logout", authHandler.Logout)
	mux.Handle("/api/v1/", authMiddleware.Authenticate(protectedMux))

	return httpx.Wrap(mux, logger, cfg.AllowedOrigin), nil
}
