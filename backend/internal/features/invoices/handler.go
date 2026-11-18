package invoices

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/portfolio/ledgerops/backend/internal/platform/httpx"
	"github.com/portfolio/ledgerops/backend/internal/platform/identity"
)

type InvoiceService interface {
	List(context.Context, string, ListParams) (Page, error)
	ListClients(context.Context, string) ([]ClientOption, error)
	Create(context.Context, string, string, MutationInput) (Invoice, error)
	Update(context.Context, string, string, string, MutationInput) (Invoice, error)
	ListAuditEvents(context.Context, string, string) ([]AuditEvent, error)
	ListPayments(context.Context, string, string) ([]Payment, error)
	RecordPayment(context.Context, string, string, Actor, PaymentInput) (PaymentResult, error)
}

type Handler struct {
	service InvoiceService
	logger  *slog.Logger
}

func NewHandler(service InvoiceService, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := httpx.OrganizationID(w, r)
	if !ok {
		return
	}
	params, validationMessage := listParams(r)
	if validationMessage != "" {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid_filter", validationMessage)
		return
	}
	page, err := h.service.List(r.Context(), organizationID, params)
	if err != nil {
		httpx.HandleApplicationError(w, h.logger, "list invoices", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

func (h *Handler) ListClients(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := httpx.OrganizationID(w, r)
	if !ok {
		return
	}
	clients, err := h.service.ListClients(r.Context(), organizationID)
	if err != nil {
		httpx.HandleApplicationError(w, h.logger, "list invoice clients", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": clients})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	organizationID, actor, ok := requestIdentity(w, r)
	if !ok {
		return
	}
	input, ok := decodeMutation(w, r)
	if !ok {
		return
	}
	invoice, err := h.service.Create(r.Context(), organizationID, actor, input)
	if err != nil {
		if writeValidationError(w, "invalid_invoice", err) {
			return
		}
		httpx.HandleApplicationError(w, h.logger, "create invoice", err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, invoice)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	organizationID, actor, ok := requestIdentity(w, r)
	if !ok {
		return
	}
	invoiceID := r.PathValue("invoiceID")
	if !httpx.ValidUUID(invoiceID) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_invoice_id", "The invoice ID is invalid.")
		return
	}
	input, ok := decodeMutation(w, r)
	if !ok {
		return
	}
	invoice, err := h.service.Update(r.Context(), organizationID, invoiceID, actor, input)
	if err != nil {
		if writeValidationError(w, "invalid_invoice", err) {
			return
		}
		httpx.HandleApplicationError(w, h.logger, "update invoice", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, invoice)
}

func (h *Handler) ListAuditEvents(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := httpx.OrganizationID(w, r)
	if !ok {
		return
	}
	invoiceID := r.PathValue("invoiceID")
	if !httpx.ValidUUID(invoiceID) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_invoice_id", "The invoice ID is invalid.")
		return
	}
	events, err := h.service.ListAuditEvents(r.Context(), organizationID, invoiceID)
	if err != nil {
		httpx.HandleApplicationError(w, h.logger, "list invoice audit events", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": events})
}

func (h *Handler) ListPayments(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := httpx.OrganizationID(w, r)
	if !ok {
		return
	}
	invoiceID := r.PathValue("invoiceID")
	if !httpx.ValidUUID(invoiceID) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_invoice_id", "The invoice ID is invalid.")
		return
	}
	payments, err := h.service.ListPayments(r.Context(), organizationID, invoiceID)
	if err != nil {
		httpx.HandleApplicationError(w, h.logger, "list invoice payments", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": payments})
}

func (h *Handler) RecordPayment(w http.ResponseWriter, r *http.Request) {
	principal, ok := authenticatedPrincipal(w, r)
	if !ok {
		return
	}
	invoiceID := r.PathValue("invoiceID")
	if !httpx.ValidUUID(invoiceID) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_invoice_id", "The invoice ID is invalid.")
		return
	}
	var input PaymentInput
	if err := httpx.DecodeJSON(w, r, &input, 8_192); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Provide a valid payment JSON body.")
		return
	}
	result, err := h.service.RecordPayment(r.Context(), principal.OrganizationID, invoiceID, Actor{
		UserID: principal.UserID,
		Email:  principal.Email,
	}, input)
	if err != nil {
		if writeValidationError(w, "invalid_payment", err) {
			return
		}
		httpx.HandleApplicationError(w, h.logger, "record invoice payment", err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, result)
}

func listParams(r *http.Request) (ListParams, string) {
	params := ListParams{
		Query:    strings.TrimSpace(r.URL.Query().Get("query")),
		Status:   strings.TrimSpace(strings.ToLower(r.URL.Query().Get("status"))),
		Page:     1,
		PageSize: DefaultPageSize,
	}
	if len(params.Query) > 100 {
		return ListParams{}, "Search terms must contain 100 characters or fewer."
	}
	if params.Status == "all" {
		params.Status = ""
	}
	if params.Status != "" && !validStatus(params.Status) {
		return ListParams{}, "Select a valid invoice status."
	}
	if raw := r.URL.Query().Get("page"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			return ListParams{}, "Page must be a positive number."
		}
		params.Page = value
	}
	if raw := r.URL.Query().Get("pageSize"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > MaxPageSize {
			return ListParams{}, "Page size must be between 1 and 50."
		}
		params.PageSize = value
	}
	return params, ""
}

func decodeMutation(w http.ResponseWriter, r *http.Request) (MutationInput, bool) {
	var input MutationInput
	if err := httpx.DecodeJSON(w, r, &input, 16_384); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Provide a valid invoice JSON body.")
		return MutationInput{}, false
	}
	return input, true
}

func writeValidationError(w http.ResponseWriter, code string, err error) bool {
	var validation ValidationError
	if !errors.As(err, &validation) {
		return false
	}
	httpx.WriteError(w, http.StatusUnprocessableEntity, code, validation.Message)
	return true
}

func requestIdentity(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	principal, ok := authenticatedPrincipal(w, r)
	if !ok {
		return "", "", false
	}
	return principal.OrganizationID, principal.Email, true
}

func authenticatedPrincipal(w http.ResponseWriter, r *http.Request) (identity.Principal, bool) {
	principal, ok := identity.FromContext(r.Context())
	if !ok || !httpx.ValidUUID(principal.OrganizationID) || !httpx.ValidUUID(principal.UserID) {
		httpx.WriteError(w, http.StatusUnauthorized, "authentication_required", "Sign in to continue.")
		return identity.Principal{}, false
	}
	return principal, true
}
