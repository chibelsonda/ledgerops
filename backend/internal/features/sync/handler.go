package sync

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/portfolio/ledgerops/backend/internal/platform/httpx"
)

type SyncService interface {
	ListRuns(context.Context, string) ([]Run, error)
	Enqueue(context.Context, string) (Run, error)
}

type Handler struct {
	service SyncService
	logger  *slog.Logger
}

func NewHandler(service SyncService, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := httpx.OrganizationID(w, r)
	if !ok {
		return
	}
	runs, err := h.service.ListRuns(r.Context(), organizationID)
	if err != nil {
		httpx.HandleApplicationError(w, h.logger, "list sync runs", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": runs})
}

func (h *Handler) Enqueue(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := httpx.OrganizationID(w, r)
	if !ok {
		return
	}
	run, err := h.service.Enqueue(r.Context(), organizationID)
	if err != nil {
		if errors.Is(err, ErrProviderNotConnected) {
			httpx.WriteError(w, http.StatusConflict, "provider_not_connected", "Connect a QuickBooks sandbox before starting a synchronization.")
			return
		}
		httpx.HandleApplicationError(w, h.logger, "enqueue sync", err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, run)
}
