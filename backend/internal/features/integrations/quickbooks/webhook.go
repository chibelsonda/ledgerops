package quickbooks

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	ledgersync "github.com/portfolio/ledgerops/backend/internal/features/sync"
	"github.com/portfolio/ledgerops/backend/internal/platform/httpx"
)

const maximumWebhookBody = 2 << 20

type WebhookEvent struct {
	EventID    string
	RealmID    string
	EntityType string
	EntityID   string
	Operation  string
	OccurredAt time.Time
	Payload    json.RawMessage
}

type WebhookRepository interface {
	StoreWebhookEvents(context.Context, []WebhookEvent) ([]string, error)
}

type SyncEnqueuer interface {
	Enqueue(context.Context, string) (ledgersync.Run, error)
}

type WebhookHandler struct {
	repository WebhookRepository
	enqueuer   SyncEnqueuer
	verifier   string
	logger     *slog.Logger
}

func NewWebhookHandler(repository WebhookRepository, enqueuer SyncEnqueuer, verifier string, logger *slog.Logger) *WebhookHandler {
	return &WebhookHandler{repository: repository, enqueuer: enqueuer, verifier: verifier, logger: logger}
}

func (h *WebhookHandler) Receive(w http.ResponseWriter, r *http.Request) {
	if h.verifier == "" {
		httpx.WriteError(w, http.StatusServiceUnavailable, "webhook_not_configured", "QuickBooks webhooks are not configured.")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maximumWebhookBody+1))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_webhook", "The webhook body could not be read.")
		return
	}
	if len(body) > maximumWebhookBody {
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "webhook_too_large", "The webhook body is too large.")
		return
	}
	if !validWebhookSignature(body, r.Header.Get("intuit-signature"), h.verifier) {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_webhook_signature", "The webhook signature is invalid.")
		return
	}
	events, err := decodeWebhookEvents(body)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_webhook", "The webhook payload is invalid.")
		return
	}
	organizationIDs, err := h.repository.StoreWebhookEvents(r.Context(), events)
	if err != nil {
		h.logger.Error("store QuickBooks webhook", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "webhook_store_failed", "The webhook could not be accepted.")
		return
	}
	for _, organizationID := range organizationIDs {
		if _, err := h.enqueuer.Enqueue(r.Context(), organizationID); err != nil {
			h.logger.Error("enqueue QuickBooks webhook sync", "organization_id", organizationID, "error", err)
		}
	}
	w.WriteHeader(http.StatusOK)
}

func validWebhookSignature(body []byte, signature, verifier string) bool {
	provided, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(verifier))
	_, _ = mac.Write(body)
	return hmac.Equal(provided, mac.Sum(nil))
}

func decodeWebhookEvents(body []byte) ([]WebhookEvent, error) {
	var cloudEvents []struct {
		ID       string          `json:"id"`
		Type     string          `json:"type"`
		Time     time.Time       `json:"time"`
		EntityID string          `json:"intuitentityid"`
		RealmID  string          `json:"intuitaccountid"`
		Data     json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &cloudEvents); err == nil && cloudEvents != nil {
		events := make([]WebhookEvent, 0, len(cloudEvents))
		for _, event := range cloudEvents {
			parts := strings.Split(event.Type, ".")
			if len(parts) < 2 || event.ID == "" || event.RealmID == "" || event.Time.IsZero() {
				return nil, fmt.Errorf("invalid QuickBooks CloudEvent")
			}
			entityType, ok := "", false
			for _, part := range parts {
				if entityType, ok = canonicalEntityType(part); ok {
					break
				}
			}
			if !ok {
				continue
			}
			events = append(events, WebhookEvent{EventID: event.ID, RealmID: event.RealmID,
				EntityType: entityType, EntityID: event.EntityID, Operation: strings.ToLower(parts[len(parts)-1]),
				OccurredAt: event.Time, Payload: event.Data})
		}
		return events, nil
	}

	var legacy struct {
		Notifications []struct {
			RealmID string `json:"realmId"`
			Change  struct {
				Entities []struct {
					Name        string `json:"name"`
					ID          string `json:"id"`
					Operation   string `json:"operation"`
					LastUpdated string `json:"lastUpdated"`
					DeletedID   string `json:"deletedId"`
				} `json:"entities"`
			} `json:"dataChangeEvent"`
		} `json:"eventNotifications"`
	}
	if err := json.Unmarshal(body, &legacy); err != nil {
		return nil, err
	}
	events := make([]WebhookEvent, 0)
	for _, notification := range legacy.Notifications {
		for _, entity := range notification.Change.Entities {
			entityType, ok := canonicalEntityType(entity.Name)
			if !ok {
				continue
			}
			occurredAt, err := time.Parse("2006-01-02T15:04:05-0700", entity.LastUpdated)
			if err != nil {
				occurredAt, err = time.Parse(time.RFC3339, entity.LastUpdated)
			}
			if err != nil {
				return nil, fmt.Errorf("parse QuickBooks webhook timestamp: %w", err)
			}
			identity := strings.Join([]string{notification.RealmID, entityType, entity.ID, entity.Operation, entity.LastUpdated}, ":")
			hash := sha256.Sum256([]byte(identity))
			events = append(events, WebhookEvent{EventID: hex.EncodeToString(hash[:]), RealmID: notification.RealmID,
				EntityType: entityType, EntityID: entity.ID, Operation: strings.ToLower(entity.Operation), OccurredAt: occurredAt,
				Payload: json.RawMessage(`{}`)})
		}
	}
	return events, nil
}

func canonicalEntityType(value string) (string, bool) {
	normalized := strings.ToLower(strings.ReplaceAll(value, "_", ""))
	for _, entityType := range ImportEntityTypes {
		if strings.ToLower(entityType) == normalized {
			return entityType, true
		}
	}
	return "", false
}
