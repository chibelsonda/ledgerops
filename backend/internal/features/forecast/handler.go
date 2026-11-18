package forecast

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/portfolio/ledgerops/backend/internal/platform/httpx"
	"github.com/portfolio/ledgerops/backend/internal/platform/identity"
)

type ForecastService interface {
	ListScenarios(context.Context, string) ([]Scenario, error)
	Get(context.Context, string, string) (Forecast, error)
	CreateScenario(context.Context, string, Actor, ScenarioInput) (Forecast, error)
	AddAdjustment(context.Context, string, string, Actor, AdjustmentInput) (Forecast, error)
}

type Handler struct {
	service ForecastService
	logger  *slog.Logger
}

func NewHandler(service ForecastService, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) ListScenarios(w http.ResponseWriter, r *http.Request) {
	p, ok := forecastPrincipal(w, r)
	if !ok {
		return
	}
	items, err := h.service.ListScenarios(r.Context(), p.OrganizationID)
	if err != nil {
		httpx.HandleApplicationError(w, h.logger, "list cash-flow scenarios", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	p, ok := forecastPrincipal(w, r)
	if !ok {
		return
	}
	id, ok := forecastID(w, r, "scenarioID", "scenario")
	if !ok {
		return
	}
	result, err := h.service.Get(r.Context(), p.OrganizationID, id)
	if err != nil {
		httpx.HandleApplicationError(w, h.logger, "get cash-flow forecast", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}
func (h *Handler) CreateScenario(w http.ResponseWriter, r *http.Request) {
	p, ok := forecastPrincipal(w, r)
	if !ok {
		return
	}
	var input ScenarioInput
	if !forecastDecode(w, r, &input) {
		return
	}
	result, err := h.service.CreateScenario(r.Context(), p.OrganizationID, forecastActor(p), input)
	if err != nil {
		if forecastValidation(w, err) {
			return
		}
		httpx.HandleApplicationError(w, h.logger, "create cash-flow scenario", err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, result)
}
func (h *Handler) AddAdjustment(w http.ResponseWriter, r *http.Request) {
	p, ok := forecastPrincipal(w, r)
	if !ok {
		return
	}
	id, ok := forecastID(w, r, "scenarioID", "scenario")
	if !ok {
		return
	}
	var input AdjustmentInput
	if !forecastDecode(w, r, &input) {
		return
	}
	result, err := h.service.AddAdjustment(r.Context(), p.OrganizationID, id, forecastActor(p), input)
	if err != nil {
		if forecastValidation(w, err) {
			return
		}
		httpx.HandleApplicationError(w, h.logger, "add cash-flow adjustment", err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, result)
}

func forecastPrincipal(w http.ResponseWriter, r *http.Request) (identity.Principal, bool) {
	p, ok := identity.FromContext(r.Context())
	if !ok || !httpx.ValidUUID(p.OrganizationID) || !httpx.ValidUUID(p.UserID) {
		httpx.WriteError(w, http.StatusUnauthorized, "authentication_required", "Sign in to continue.")
		return identity.Principal{}, false
	}
	return p, true
}
func forecastActor(p identity.Principal) Actor { return Actor{UserID: p.UserID, Email: p.Email} }
func forecastID(w http.ResponseWriter, r *http.Request, name, label string) (string, bool) {
	id := r.PathValue(name)
	if !httpx.ValidUUID(id) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_"+name, "The "+label+" ID is invalid.")
		return "", false
	}
	return id, true
}
func forecastDecode(w http.ResponseWriter, r *http.Request, destination any) bool {
	if httpx.DecodeJSON(w, r, destination, 16_384) != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Provide a valid cash-flow forecast JSON body.")
		return false
	}
	return true
}
func forecastValidation(w http.ResponseWriter, err error) bool {
	var validation ValidationError
	if !errors.As(err, &validation) {
		return false
	}
	httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid_cash_flow_forecast", validation.Message)
	return true
}
