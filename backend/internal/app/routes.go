package app

import (
	"log/slog"
	"net/http"

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
	"github.com/portfolio/ledgerops/backend/internal/platform/identity"
)

type routeHandlers struct {
	auth           *auth.Handler
	dashboard      *dashboard.Handler
	invoices       *invoices.Handler
	adjustments    *adjustments.Handler
	clients        *clients.Handler
	projects       *projects.Handler
	close          *closefeature.Handler
	forecast       *forecast.Handler
	reminders      *reminders.Handler
	reconciliation *reconciliation.Handler
	sync           *sync.Handler
	quickBooks     *quickbooks.Handler
}

func registerProtectedRoutes(mux *http.ServeMux, logger *slog.Logger, handlers routeHandlers) {
	allRoles := auth.RequireRoles(logger, identity.RoleAdmin, identity.RoleController, identity.RoleReviewer)
	operators := auth.RequireRoles(logger, identity.RoleAdmin, identity.RoleController)
	admins := auth.RequireRoles(logger, identity.RoleAdmin)
	controllers := auth.RequireRoles(logger, identity.RoleController)

	mux.Handle("GET /api/v1/dashboard", allRoles(http.HandlerFunc(handlers.dashboard.Get)))
	mux.Handle("GET /api/v1/invoices", allRoles(http.HandlerFunc(handlers.invoices.List)))
	mux.Handle("GET /api/v1/invoices/clients", allRoles(http.HandlerFunc(handlers.invoices.ListClients)))
	mux.Handle("POST /api/v1/invoices", operators(http.HandlerFunc(handlers.invoices.Create)))
	mux.Handle("PATCH /api/v1/invoices/{invoiceID}", operators(http.HandlerFunc(handlers.invoices.Update)))
	mux.Handle("GET /api/v1/invoices/{invoiceID}/audit-events", allRoles(http.HandlerFunc(handlers.invoices.ListAuditEvents)))
	mux.Handle("GET /api/v1/invoices/{invoiceID}/payments", allRoles(http.HandlerFunc(handlers.invoices.ListPayments)))
	mux.Handle("POST /api/v1/invoices/{invoiceID}/payments", operators(http.HandlerFunc(handlers.invoices.RecordPayment)))
	mux.Handle("GET /api/v1/invoices/{invoiceID}/adjustments", allRoles(http.HandlerFunc(handlers.adjustments.List)))
	mux.Handle("POST /api/v1/invoices/{invoiceID}/adjustments", operators(http.HandlerFunc(handlers.adjustments.Create)))
	mux.Handle("PATCH /api/v1/adjustments/{adjustmentID}", admins(http.HandlerFunc(handlers.adjustments.Decide)))
	mux.Handle("GET /api/v1/clients", allRoles(http.HandlerFunc(handlers.clients.List)))
	mux.Handle("POST /api/v1/clients", operators(http.HandlerFunc(handlers.clients.Create)))
	mux.Handle("PATCH /api/v1/clients/{clientID}", operators(http.HandlerFunc(handlers.clients.Update)))
	mux.Handle("GET /api/v1/clients/{clientID}/contacts", allRoles(http.HandlerFunc(handlers.clients.ListContacts)))
	mux.Handle("POST /api/v1/clients/{clientID}/contacts", operators(http.HandlerFunc(handlers.clients.CreateContact)))
	mux.Handle("PATCH /api/v1/contacts/{contactID}", operators(http.HandlerFunc(handlers.clients.UpdateContact)))
	mux.Handle("GET /api/v1/projects", allRoles(http.HandlerFunc(handlers.projects.List)))
	mux.Handle("GET /api/v1/projects/clients", allRoles(http.HandlerFunc(handlers.projects.ListClients)))
	mux.Handle("GET /api/v1/projects/{projectID}", allRoles(http.HandlerFunc(handlers.projects.Get)))
	mux.Handle("POST /api/v1/projects", operators(http.HandlerFunc(handlers.projects.Create)))
	mux.Handle("PATCH /api/v1/contracts/{contractID}/decision", admins(http.HandlerFunc(handlers.projects.DecideContract)))
	mux.Handle("POST /api/v1/contracts/{contractID}/change-orders", operators(http.HandlerFunc(handlers.projects.CreateChangeOrder)))
	mux.Handle("PATCH /api/v1/change-orders/{changeOrderID}/decision", admins(http.HandlerFunc(handlers.projects.DecideChangeOrder)))
	mux.Handle("POST /api/v1/contracts/{contractID}/milestones", operators(http.HandlerFunc(handlers.projects.CreateMilestone)))
	mux.Handle("POST /api/v1/milestones/{milestoneID}/invoice", operators(http.HandlerFunc(handlers.projects.InvoiceMilestone)))
	mux.Handle("GET /api/v1/close-periods", allRoles(http.HandlerFunc(handlers.close.List)))
	mux.Handle("GET /api/v1/close-periods/{periodID}", allRoles(http.HandlerFunc(handlers.close.Get)))
	mux.Handle("POST /api/v1/close-periods", controllers(http.HandlerFunc(handlers.close.Create)))
	mux.Handle("PATCH /api/v1/close-tasks/{taskID}", operators(http.HandlerFunc(handlers.close.UpdateTask)))
	mux.Handle("POST /api/v1/close-periods/{periodID}/submit", operators(http.HandlerFunc(handlers.close.Submit)))
	mux.Handle("PATCH /api/v1/close-periods/{periodID}/decision", admins(http.HandlerFunc(handlers.close.Decide)))
	mux.Handle("GET /api/v1/cash-flow/scenarios", allRoles(http.HandlerFunc(handlers.forecast.ListScenarios)))
	mux.Handle("GET /api/v1/cash-flow/scenarios/{scenarioID}", allRoles(http.HandlerFunc(handlers.forecast.Get)))
	mux.Handle("POST /api/v1/cash-flow/scenarios", operators(http.HandlerFunc(handlers.forecast.CreateScenario)))
	mux.Handle("POST /api/v1/cash-flow/scenarios/{scenarioID}/adjustments", operators(http.HandlerFunc(handlers.forecast.AddAdjustment)))
	mux.Handle("GET /api/v1/reminders", allRoles(http.HandlerFunc(handlers.reminders.List)))
	mux.Handle("GET /api/v1/reminders/delivery-status", allRoles(http.HandlerFunc(handlers.reminders.DeliveryStatus)))
	mux.Handle("POST /api/v1/reminders/generate", operators(http.HandlerFunc(handlers.reminders.Generate)))
	mux.Handle("PATCH /api/v1/reminders/{reminderID}/cancel", operators(http.HandlerFunc(handlers.reminders.Cancel)))
	mux.Handle("GET /api/v1/reconciliation", allRoles(http.HandlerFunc(handlers.reconciliation.List)))
	mux.Handle("PATCH /api/v1/reconciliation/{itemID}", allRoles(http.HandlerFunc(handlers.reconciliation.Resolve)))
	mux.Handle("GET /api/v1/sync-runs", operators(http.HandlerFunc(handlers.sync.List)))
	mux.Handle("POST /api/v1/sync-runs", operators(http.HandlerFunc(handlers.sync.Enqueue)))
	mux.Handle("GET /api/v1/integrations/quickbooks", operators(http.HandlerFunc(handlers.quickBooks.Status)))
	mux.Handle("POST /api/v1/integrations/quickbooks/connect", admins(http.HandlerFunc(handlers.quickBooks.Connect)))
	mux.Handle("DELETE /api/v1/integrations/quickbooks", admins(http.HandlerFunc(handlers.quickBooks.Disconnect)))
	mux.HandleFunc("POST /api/v1/auth/logout", handlers.auth.Logout)
}
