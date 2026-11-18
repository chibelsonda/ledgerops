package quickbooks

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/portfolio/ledgerops/backend/internal/platform/httpx"
)

type IntegrationService interface {
	Status(context.Context, string) (ConnectionStatus, error)
	BeginAuthorization(context.Context, string) (Authorization, error)
	CompleteAuthorization(context.Context, string, string, string) error
	Disconnect(context.Context, string) error
}

type Handler struct {
	service     IntegrationService
	logger      *slog.Logger
	frontendURL string
}

func NewHandler(service IntegrationService, logger *slog.Logger, frontendURL string) *Handler {
	return &Handler{service: service, logger: logger, frontendURL: strings.TrimRight(frontendURL, "/")}
}

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := httpx.OrganizationID(w, r)
	if !ok {
		return
	}
	status, err := h.service.Status(r.Context(), organizationID)
	if err != nil {
		handleError(w, h.logger, "get QuickBooks connection", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, status)
}

func (h *Handler) Connect(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := httpx.OrganizationID(w, r)
	if !ok {
		return
	}
	authorization, err := h.service.BeginAuthorization(r.Context(), organizationID)
	if err != nil {
		handleError(w, h.logger, "begin QuickBooks authorization", err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, authorization)
}

func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if providerError := query.Get("error"); providerError != "" {
		h.logger.Warn("QuickBooks authorization denied", "provider_error", providerError)
		h.redirect(w, r, "error", "QuickBooks access was not granted.")
		return
	}
	if err := h.service.CompleteAuthorization(r.Context(), query.Get("state"), query.Get("code"), query.Get("realmId")); err != nil {
		h.logger.Error("complete QuickBooks authorization", "error", err)
		message := "QuickBooks could not be connected. Start the connection again."
		if errors.Is(err, ErrInvalidState) {
			message = "The QuickBooks connection request expired. Please try again."
		}
		h.redirect(w, r, "error", message)
		return
	}
	h.redirect(w, r, "connected", "")
}

func (h *Handler) Disconnect(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := httpx.OrganizationID(w, r)
	if !ok {
		return
	}
	if err := h.service.Disconnect(r.Context(), organizationID); err != nil {
		handleError(w, h.logger, "disconnect QuickBooks", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) redirect(w http.ResponseWriter, r *http.Request, result, message string) {
	destination, err := url.Parse(h.frontendURL + "/sync")
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "invalid_frontend_url", "The frontend URL is invalid.")
		return
	}
	query := destination.Query()
	query.Set("quickbooks", result)
	if message != "" {
		query.Set("message", message)
	}
	destination.RawQuery = query.Encode()
	http.Redirect(w, r, destination.String(), http.StatusSeeOther)
}

func handleError(w http.ResponseWriter, logger *slog.Logger, operation string, err error) {
	if errors.Is(err, ErrNotConfigured) {
		httpx.WriteError(w, http.StatusServiceUnavailable, "integration_not_configured", "QuickBooks sandbox credentials are not configured.")
		return
	}
	if errors.Is(err, ErrInvalidCallback) || errors.Is(err, ErrInvalidState) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_oauth_callback", "The QuickBooks authorization response is invalid or expired.")
		return
	}
	logger.Error(operation, "error", err)
	httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "The request could not be completed.")
}
