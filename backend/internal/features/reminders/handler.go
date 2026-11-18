package reminders

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"

	"github.com/portfolio/ledgerops/backend/internal/platform/httpx"
)

type ReminderService interface {
	List(context.Context, string) ([]Reminder, error)
	GenerateDue(context.Context, string) (GenerateResult, error)
	Cancel(context.Context, string, string, Actor) error
	RecordDeliveryEvent(context.Context, DeliveryEventInput) error
}

type Handler struct {
	service       ReminderService
	logger        *slog.Logger
	status        DeliveryStatus
	webhookSecret string
}

func NewHandler(service ReminderService, logger *slog.Logger, status DeliveryStatus, webhookSecret string) *Handler {
	return &Handler{service: service, logger: logger, status: status, webhookSecret: webhookSecret}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpx.AuthenticatedPrincipal(w, r)
	if !ok {
		return
	}
	items, err := h.service.List(r.Context(), principal.OrganizationID)
	if err != nil {
		httpx.HandleApplicationError(w, h.logger, "list reminders", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) DeliveryStatus(w http.ResponseWriter, r *http.Request) {
	if _, ok := httpx.AuthenticatedPrincipal(w, r); !ok {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, h.status)
}

func (h *Handler) DeliveryWebhook(w http.ResponseWriter, r *http.Request) {
	if h.webhookSecret == "" {
		httpx.WriteError(w, http.StatusServiceUnavailable, "webhook_not_configured", "Email delivery webhooks are not configured.")
		return
	}
	if !sameSecret(r.Header.Get("X-LedgerOps-Webhook-Secret"), h.webhookSecret) {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_webhook_secret", "The webhook secret is invalid.")
		return
	}
	var input DeliveryEventInput
	if err := httpx.DecodeJSON(w, r, &input, 64<<10); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "Provide a valid delivery event payload.")
		return
	}
	if err := h.service.RecordDeliveryEvent(r.Context(), input); err != nil {
		if errors.Is(err, ErrInvalidDeliveryEvent) {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_delivery_event", "The delivery event is invalid.")
			return
		}
		httpx.HandleApplicationError(w, h.logger, "record reminder delivery event", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func sameSecret(provided, expected string) bool {
	providedDigest := sha256.Sum256([]byte(provided))
	expectedDigest := sha256.Sum256([]byte(expected))
	return subtle.ConstantTimeCompare(providedDigest[:], expectedDigest[:]) == 1
}

func (h *Handler) Generate(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpx.AuthenticatedPrincipal(w, r)
	if !ok {
		return
	}
	result, err := h.service.GenerateDue(r.Context(), principal.OrganizationID)
	if err != nil {
		httpx.HandleApplicationError(w, h.logger, "generate reminders", err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, result)
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpx.AuthenticatedPrincipal(w, r)
	if !ok {
		return
	}
	id := r.PathValue("reminderID")
	if !httpx.ValidUUID(id) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_reminder_id", "The reminder ID is invalid.")
		return
	}
	if err := h.service.Cancel(r.Context(), principal.OrganizationID, id, Actor{UserID: principal.UserID, Email: principal.Email}); err != nil {
		httpx.HandleApplicationError(w, h.logger, "cancel reminder", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
