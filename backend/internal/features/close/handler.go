package close

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/portfolio/ledgerops/backend/internal/platform/httpx"
	"github.com/portfolio/ledgerops/backend/internal/platform/identity"
)

type CloseService interface {
	List(context.Context, string) ([]PeriodSummary, error)
	Get(context.Context, string, string) (Detail, error)
	Create(context.Context, string, Actor, CreateInput) (Detail, error)
	UpdateTask(context.Context, string, string, Actor, TaskInput) (Detail, error)
	Submit(context.Context, string, string, Actor) (Detail, error)
	Decide(context.Context, string, string, Actor, DecisionInput) (Detail, error)
}

type Handler struct {
	service CloseService
	logger  *slog.Logger
}

func NewHandler(service CloseService, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	p, ok := httpx.AuthenticatedPrincipal(w, r)
	if !ok {
		return
	}
	items, err := h.service.List(r.Context(), p.OrganizationID)
	if err != nil {
		httpx.HandleApplicationError(w, h.logger, "list close periods", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	p, ok := httpx.AuthenticatedPrincipal(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "periodID", "period")
	if !ok {
		return
	}
	detail, err := h.service.Get(r.Context(), p.OrganizationID, id)
	if err != nil {
		httpx.HandleApplicationError(w, h.logger, "get close period", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, detail)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	p, ok := httpx.AuthenticatedPrincipal(w, r)
	if !ok {
		return
	}
	var input CreateInput
	if !decode(w, r, &input) {
		return
	}
	detail, err := h.service.Create(r.Context(), p.OrganizationID, actor(p), input)
	if err != nil {
		if httpx.WriteValidationError(w, "invalid_month_end_close", err) {
			return
		}
		httpx.HandleApplicationError(w, h.logger, "create close period", err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, detail)
}

func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	p, ok := httpx.AuthenticatedPrincipal(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "taskID", "task")
	if !ok {
		return
	}
	var input TaskInput
	if !decode(w, r, &input) {
		return
	}
	detail, err := h.service.UpdateTask(r.Context(), p.OrganizationID, id, actor(p), input)
	if err != nil {
		if httpx.WriteValidationError(w, "invalid_month_end_close", err) {
			return
		}
		httpx.HandleApplicationError(w, h.logger, "update close task", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, detail)
}

func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	p, ok := httpx.AuthenticatedPrincipal(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "periodID", "period")
	if !ok {
		return
	}
	detail, err := h.service.Submit(r.Context(), p.OrganizationID, id, actor(p))
	if err != nil {
		httpx.HandleApplicationError(w, h.logger, "submit close period", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, detail)
}

func (h *Handler) Decide(w http.ResponseWriter, r *http.Request) {
	p, ok := httpx.AuthenticatedPrincipal(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "periodID", "period")
	if !ok {
		return
	}
	var input DecisionInput
	if !decode(w, r, &input) {
		return
	}
	detail, err := h.service.Decide(r.Context(), p.OrganizationID, id, actor(p), input)
	if err != nil {
		if httpx.WriteValidationError(w, "invalid_month_end_close", err) {
			return
		}
		httpx.HandleApplicationError(w, h.logger, "decide close period", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, detail)
}

func actor(p identity.Principal) Actor { return Actor{UserID: p.UserID, Email: p.Email} }
func pathID(w http.ResponseWriter, r *http.Request, name, label string) (string, bool) {
	id := r.PathValue(name)
	if !httpx.ValidUUID(id) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_"+name, "The "+label+" ID is invalid.")
		return "", false
	}
	return id, true
}
func decode(w http.ResponseWriter, r *http.Request, destination any) bool {
	if httpx.DecodeJSON(w, r, destination, 16_384) != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Provide a valid month-end close JSON body.")
		return false
	}
	return true
}
