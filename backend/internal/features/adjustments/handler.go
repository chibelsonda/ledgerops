package adjustments

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/portfolio/ledgerops/backend/internal/platform/httpx"
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
	principal, ok := httpx.AuthenticatedPrincipal(w, r)
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
	principal, ok := httpx.AuthenticatedPrincipal(w, r)
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
		if httpx.WriteValidationError(w, "invalid_adjustment", err) {
			return
		}
		httpx.HandleApplicationError(w, h.logger, "create invoice adjustment", err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, result)
}

func (h *Handler) Decide(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpx.AuthenticatedPrincipal(w, r)
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
		if httpx.WriteValidationError(w, "invalid_decision", err) {
			return
		}
		httpx.HandleApplicationError(w, h.logger, "decide invoice adjustment", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}
