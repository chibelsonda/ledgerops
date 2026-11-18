package projects

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/portfolio/ledgerops/backend/internal/platform/httpx"
	"github.com/portfolio/ledgerops/backend/internal/platform/identity"
)

type ProjectService interface {
	List(context.Context, string) ([]ProjectSummary, error)
	ListClients(context.Context, string) ([]ClientOption, error)
	Get(context.Context, string, string) (Detail, error)
	Create(context.Context, string, Actor, CreateInput) (Detail, error)
	DecideContract(context.Context, string, string, Actor, DecisionInput) (Detail, error)
	CreateChangeOrder(context.Context, string, string, Actor, ChangeOrderInput) (Detail, error)
	DecideChangeOrder(context.Context, string, string, Actor, DecisionInput) (Detail, error)
	CreateMilestone(context.Context, string, string, Actor, MilestoneInput) (Detail, error)
	InvoiceMilestone(context.Context, string, string, Actor, InvoiceInput) (Detail, error)
}

type Handler struct {
	service ProjectService
	logger  *slog.Logger
}

func NewHandler(service ProjectService, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	principal, ok := projectPrincipal(w, r)
	if !ok {
		return
	}
	items, err := h.service.List(r.Context(), principal.OrganizationID)
	if err != nil {
		httpx.HandleApplicationError(w, h.logger, "list projects", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) ListClients(w http.ResponseWriter, r *http.Request) {
	principal, ok := projectPrincipal(w, r)
	if !ok {
		return
	}
	items, err := h.service.ListClients(r.Context(), principal.OrganizationID)
	if err != nil {
		httpx.HandleApplicationError(w, h.logger, "list project clients", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	principal, ok := projectPrincipal(w, r)
	if !ok {
		return
	}
	projectID, ok := projectPathID(w, r, "projectID", "project")
	if !ok {
		return
	}
	detail, err := h.service.Get(r.Context(), principal.OrganizationID, projectID)
	if err != nil {
		httpx.HandleApplicationError(w, h.logger, "get project", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, detail)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	principal, ok := projectPrincipal(w, r)
	if !ok {
		return
	}
	var input CreateInput
	if !decodeProjectJSON(w, r, &input) {
		return
	}
	detail, err := h.service.Create(r.Context(), principal.OrganizationID, projectActor(principal), input)
	if err != nil {
		if writeProjectValidationError(w, err) {
			return
		}
		httpx.HandleApplicationError(w, h.logger, "create project", err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, detail)
}

func (h *Handler) DecideContract(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, "contractID", "contract", h.service.DecideContract)
}

func (h *Handler) DecideChangeOrder(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, "changeOrderID", "change order", h.service.DecideChangeOrder)
}

type decisionUseCase func(context.Context, string, string, Actor, DecisionInput) (Detail, error)

func (h *Handler) decide(w http.ResponseWriter, r *http.Request, pathName, label string, useCase decisionUseCase) {
	principal, ok := projectPrincipal(w, r)
	if !ok {
		return
	}
	id, ok := projectPathID(w, r, pathName, label)
	if !ok {
		return
	}
	var input DecisionInput
	if !decodeProjectJSON(w, r, &input) {
		return
	}
	detail, err := useCase(r.Context(), principal.OrganizationID, id, projectActor(principal), input)
	if err != nil {
		if writeProjectValidationError(w, err) {
			return
		}
		httpx.HandleApplicationError(w, h.logger, "decide "+label, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, detail)
}

func (h *Handler) CreateChangeOrder(w http.ResponseWriter, r *http.Request) {
	principal, ok := projectPrincipal(w, r)
	if !ok {
		return
	}
	contractID, ok := projectPathID(w, r, "contractID", "contract")
	if !ok {
		return
	}
	var input ChangeOrderInput
	if !decodeProjectJSON(w, r, &input) {
		return
	}
	detail, err := h.service.CreateChangeOrder(r.Context(), principal.OrganizationID, contractID, projectActor(principal), input)
	if err != nil {
		if writeProjectValidationError(w, err) {
			return
		}
		httpx.HandleApplicationError(w, h.logger, "create change order", err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, detail)
}

func (h *Handler) CreateMilestone(w http.ResponseWriter, r *http.Request) {
	principal, ok := projectPrincipal(w, r)
	if !ok {
		return
	}
	contractID, ok := projectPathID(w, r, "contractID", "contract")
	if !ok {
		return
	}
	var input MilestoneInput
	if !decodeProjectJSON(w, r, &input) {
		return
	}
	detail, err := h.service.CreateMilestone(r.Context(), principal.OrganizationID, contractID, projectActor(principal), input)
	if err != nil {
		if writeProjectValidationError(w, err) {
			return
		}
		httpx.HandleApplicationError(w, h.logger, "create billing milestone", err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, detail)
}

func (h *Handler) InvoiceMilestone(w http.ResponseWriter, r *http.Request) {
	principal, ok := projectPrincipal(w, r)
	if !ok {
		return
	}
	milestoneID, ok := projectPathID(w, r, "milestoneID", "milestone")
	if !ok {
		return
	}
	var input InvoiceInput
	if !decodeProjectJSON(w, r, &input) {
		return
	}
	detail, err := h.service.InvoiceMilestone(r.Context(), principal.OrganizationID, milestoneID, projectActor(principal), input)
	if err != nil {
		if writeProjectValidationError(w, err) {
			return
		}
		httpx.HandleApplicationError(w, h.logger, "invoice billing milestone", err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, detail)
}

func decodeProjectJSON(w http.ResponseWriter, r *http.Request, destination any) bool {
	if err := httpx.DecodeJSON(w, r, destination, 16_384); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Provide a valid project billing JSON body.")
		return false
	}
	return true
}

func writeProjectValidationError(w http.ResponseWriter, err error) bool {
	var validation ValidationError
	if !errors.As(err, &validation) {
		return false
	}
	httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid_project_billing", validation.Message)
	return true
}

func projectPathID(w http.ResponseWriter, r *http.Request, name, label string) (string, bool) {
	id := r.PathValue(name)
	if !httpx.ValidUUID(id) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_"+name, "The "+label+" ID is invalid.")
		return "", false
	}
	return id, true
}

func projectPrincipal(w http.ResponseWriter, r *http.Request) (identity.Principal, bool) {
	principal, ok := identity.FromContext(r.Context())
	if !ok || !httpx.ValidUUID(principal.OrganizationID) || !httpx.ValidUUID(principal.UserID) {
		httpx.WriteError(w, http.StatusUnauthorized, "authentication_required", "Sign in to continue.")
		return identity.Principal{}, false
	}
	return principal, true
}

func projectActor(principal identity.Principal) Actor {
	return Actor{UserID: principal.UserID, Email: principal.Email}
}
