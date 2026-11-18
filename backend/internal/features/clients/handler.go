package clients

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/portfolio/ledgerops/backend/internal/platform/httpx"
)

type ClientService interface {
	List(context.Context, string, ListParams) (Page, error)
	Create(context.Context, string, Actor, MutationInput) (Client, error)
	Update(context.Context, string, string, Actor, MutationInput) (Client, error)
	ListContacts(context.Context, string, string) ([]Contact, error)
	CreateContact(context.Context, string, string, Actor, ContactInput) (Contact, error)
	UpdateContact(context.Context, string, string, Actor, ContactInput) (Contact, error)
}

type Handler struct {
	service ClientService
	logger  *slog.Logger
}

func NewHandler(service ClientService, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpx.AuthenticatedPrincipal(w, r)
	if !ok {
		return
	}
	params, message := parseListParams(r)
	if message != "" {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid_filter", message)
		return
	}
	page, err := h.service.List(r.Context(), principal.OrganizationID, params)
	if err != nil {
		httpx.HandleApplicationError(w, h.logger, "list clients", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpx.AuthenticatedPrincipal(w, r)
	if !ok {
		return
	}
	var input MutationInput
	if err := httpx.DecodeJSON(w, r, &input, 16_384); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Provide valid client details.")
		return
	}
	result, err := h.service.Create(r.Context(), principal.OrganizationID, Actor{UserID: principal.UserID, Email: principal.Email}, input)
	if err != nil {
		if clientValidationError(w, err) {
			return
		}
		httpx.HandleApplicationError(w, h.logger, "create client", err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, result)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpx.AuthenticatedPrincipal(w, r)
	if !ok {
		return
	}
	clientID := r.PathValue("clientID")
	if !httpx.ValidUUID(clientID) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_client_id", "The client ID is invalid.")
		return
	}
	var input MutationInput
	if err := httpx.DecodeJSON(w, r, &input, 16_384); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Provide valid client details.")
		return
	}
	result, err := h.service.Update(r.Context(), principal.OrganizationID, clientID, Actor{UserID: principal.UserID, Email: principal.Email}, input)
	if err != nil {
		if clientValidationError(w, err) {
			return
		}
		httpx.HandleApplicationError(w, h.logger, "update client", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) ListContacts(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpx.AuthenticatedPrincipal(w, r)
	if !ok {
		return
	}
	clientID := r.PathValue("clientID")
	if !httpx.ValidUUID(clientID) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_client_id", "The client ID is invalid.")
		return
	}
	items, err := h.service.ListContacts(r.Context(), principal.OrganizationID, clientID)
	if err != nil {
		httpx.HandleApplicationError(w, h.logger, "list client contacts", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) CreateContact(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpx.AuthenticatedPrincipal(w, r)
	if !ok {
		return
	}
	clientID := r.PathValue("clientID")
	if !httpx.ValidUUID(clientID) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_client_id", "The client ID is invalid.")
		return
	}
	var input ContactInput
	if err := httpx.DecodeJSON(w, r, &input, 8_192); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Provide valid contact details.")
		return
	}
	result, err := h.service.CreateContact(r.Context(), principal.OrganizationID, clientID, Actor{UserID: principal.UserID, Email: principal.Email}, input)
	if err != nil {
		if clientValidationError(w, err) {
			return
		}
		httpx.HandleApplicationError(w, h.logger, "create contact", err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, result)
}

func (h *Handler) UpdateContact(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpx.AuthenticatedPrincipal(w, r)
	if !ok {
		return
	}
	contactID := r.PathValue("contactID")
	if !httpx.ValidUUID(contactID) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_contact_id", "The contact ID is invalid.")
		return
	}
	var input ContactInput
	if err := httpx.DecodeJSON(w, r, &input, 8_192); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Provide valid contact details.")
		return
	}
	result, err := h.service.UpdateContact(r.Context(), principal.OrganizationID, contactID, Actor{UserID: principal.UserID, Email: principal.Email}, input)
	if err != nil {
		if clientValidationError(w, err) {
			return
		}
		httpx.HandleApplicationError(w, h.logger, "update contact", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func parseListParams(r *http.Request) (ListParams, string) {
	params := ListParams{Query: strings.TrimSpace(r.URL.Query().Get("query")), Page: 1, PageSize: DefaultPageSize}
	if len(params.Query) > 100 {
		return ListParams{}, "Search terms must contain 100 characters or fewer."
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
		if err != nil || value < 1 || value > 50 {
			return ListParams{}, "Page size must be between 1 and 50."
		}
		params.PageSize = value
	}
	return params, ""
}

func clientValidationError(w http.ResponseWriter, err error) bool {
	var validation ValidationError
	if !errors.As(err, &validation) {
		return false
	}
	httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid_client", validation.Message)
	return true
}
