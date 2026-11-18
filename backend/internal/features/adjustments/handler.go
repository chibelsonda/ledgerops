package adjustments

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/portfolio/ledgerops/backend/internal/platform/httpx"
	"github.com/portfolio/ledgerops/backend/internal/platform/identity"
)

type AdjustmentService interface {
	List(context.Context, string, string) ([]Adjustment, error)
	Create(context.Context, string, string, Actor, CreateInput) (Adjustment, error)
	Decide(context.Context, string, string, Actor, DecisionInput) (DecisionResult, error)
}

type Handler struct {
	service AdjustmentService
	logger  *slog.Logger
}

func NewHandler(service AdjustmentService, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	principal, ok := principal(w, r)
	if !ok {
		return
	}
	invoiceID := r.PathValue("invoiceID")
	if !httpx.ValidUUID(invoiceID) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_invoice_id", "The invoice ID is invalid.")
		return
	}
	items, err := h.service.List(r.Context(), principal.OrganizationID, invoiceID)
	if err != nil {
		httpx.HandleApplicationError(w, h.logger, "list invoice adjustments", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	principal, ok := principal(w, r)
	if !ok {
		return
	}
	invoiceID := r.PathValue("invoiceID")
	if !httpx.ValidUUID(invoiceID) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_invoice_id", "The invoice ID is invalid.")
		return
	}
	var input CreateInput
	if err := httpx.DecodeJSON(w, r, &input, 8_192); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Provide a valid adjustment JSON body.")
		return
	}
	result, err := h.service.Create(r.Context(), principal.OrganizationID, invoiceID, Actor{UserID: principal.UserID, Email: principal.Email}, input)
	if err != nil {
		if writeValidationError(w, "invalid_adjustment", err) {
			return
		}
		httpx.HandleApplicationError(w, h.logger, "create invoice adjustment", err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, result)
}

func (h *Handler) Decide(w http.ResponseWriter, r *http.Request) {
	principal, ok := principal(w, r)
	if !ok {
		return
	}
	adjustmentID := r.PathValue("adjustmentID")
	if !httpx.ValidUUID(adjustmentID) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_adjustment_id", "The adjustment ID is invalid.")
		return
	}
	var input DecisionInput
	if err := httpx.DecodeJSON(w, r, &input, 8_192); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Provide a valid decision JSON body.")
		return
	}
	result, err := h.service.Decide(r.Context(), principal.OrganizationID, adjustmentID, Actor{UserID: principal.UserID, Email: principal.Email}, input)
	if err != nil {
		if writeValidationError(w, "invalid_decision", err) {
			return
		}
		httpx.HandleApplicationError(w, h.logger, "decide invoice adjustment", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func writeValidationError(w http.ResponseWriter, code string, err error) bool {
	var validation ValidationError
	if !errors.As(err, &validation) {
		return false
	}
	httpx.WriteError(w, http.StatusUnprocessableEntity, code, validation.Message)
	return true
}

func principal(w http.ResponseWriter, r *http.Request) (identity.Principal, bool) {
	value, ok := identity.FromContext(r.Context())
	if !ok || !httpx.ValidUUID(value.OrganizationID) || !httpx.ValidUUID(value.UserID) {
		httpx.WriteError(w, http.StatusUnauthorized, "authentication_required", "Sign in to continue.")
		return identity.Principal{}, false
	}
	return value, true
}
