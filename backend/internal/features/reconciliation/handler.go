package reconciliation

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/portfolio/ledgerops/backend/internal/platform/httpx"
	"github.com/portfolio/ledgerops/backend/internal/platform/identity"
)

type ReconciliationService interface {
	List(context.Context, string) ([]Item, error)
	Resolve(context.Context, string, string, string, string) error
}

type Handler struct {
	service ReconciliationService
	logger  *slog.Logger
}

func NewHandler(service ReconciliationService, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := httpx.OrganizationID(w, r)
	if !ok {
		return
	}
	items, err := h.service.List(r.Context(), organizationID)
	if err != nil {
		httpx.HandleApplicationError(w, h.logger, "list reconciliation items", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) Resolve(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := httpx.OrganizationID(w, r)
	if !ok {
		return
	}
	itemID := r.PathValue("itemID")
	if !httpx.ValidUUID(itemID) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_item_id", "The reconciliation item ID is invalid.")
		return
	}
	var request Resolution
	if err := httpx.DecodeJSON(w, r, &request, 8_192); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Provide a JSON body with a resolution note.")
		return
	}
	principal, _ := identity.FromContext(r.Context())
	if err := h.service.Resolve(r.Context(), organizationID, itemID, principal.Email, request.Note); err != nil {
		if errors.Is(err, ErrInvalidResolution) {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid_note", "The resolution note must contain 3 to 500 characters.")
			return
		}
		httpx.HandleApplicationError(w, h.logger, "resolve reconciliation item", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
