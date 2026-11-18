package dashboard

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/portfolio/ledgerops/backend/internal/platform/httpx"
)

type DashboardService interface {
	Get(context.Context, string) (View, error)
}

type Handler struct {
	service DashboardService
	logger  *slog.Logger
}

func NewHandler(service DashboardService, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := httpx.OrganizationID(w, r)
	if !ok {
		return
	}
	view, err := h.service.Get(r.Context(), organizationID)
	if err != nil {
		httpx.HandleApplicationError(w, h.logger, "load dashboard", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, view)
}
