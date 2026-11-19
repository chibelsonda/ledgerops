package reminders

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/portfolio/ledgerops/backend/internal/platform/apperrors"
	"github.com/portfolio/ledgerops/backend/internal/platform/audit"
	"gorm.io/gorm"
)

type PostgresRepository struct{ db *gorm.DB }

func NewPostgresRepository(db *gorm.DB) *PostgresRepository { return &PostgresRepository{db: db} }

func (r *PostgresRepository) List(ctx context.Context, organizationID string) ([]Reminder, error) {
	items := make([]Reminder, 0)
	if err := r.db.WithContext(ctx).Raw(`
		SELECT r.id::text AS id,r.invoice_id::text AS invoice_id,i.invoice_number,r.client_id::text AS client_id,c.name AS client_name,
			cc.name AS contact_name,r.recipient_email,r.status,to_char(r.scheduled_for,'YYYY-MM-DD') AS scheduled_for,r.subject,r.message,
			b.outstanding_minor,
			i.currency_code,to_char(r.created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at,
			r.attempts,r.max_attempts,r.last_error,r.provider_message_id,
			(SELECT e.event_type FROM reminder_delivery_events e WHERE e.reminder_id=r.id ORDER BY e.occurred_at DESC,e.received_at DESC LIMIT 1) AS delivery_event
		FROM invoice_reminders r JOIN invoices i ON i.id=r.invoice_id
		JOIN invoice_balances b ON b.invoice_id=i.id AND b.organization_id=i.organization_id
		JOIN clients c ON c.id=r.client_id JOIN client_contacts cc ON cc.id=r.contact_id
		WHERE r.organization_id=? ORDER BY CASE r.status WHEN 'queued' THEN 0 ELSE 1 END,r.scheduled_for,r.created_at DESC`, organizationID).Scan(&items).Error; err != nil {
		return nil, fmt.Errorf("list reminders: %w", err)
	}
	return items, nil
}

func (r *PostgresRepository) ClaimDelivery(ctx context.Context, lease time.Duration) (*Delivery, error) {
	var delivery Delivery
	result := r.db.WithContext(ctx).Raw(`
		WITH candidate AS (
			SELECT id FROM invoice_reminders
			WHERE status='queued' AND scheduled_for <= CURRENT_DATE AND next_attempt_at <= now()
				AND attempts < max_attempts AND (locked_at IS NULL OR locked_at < now() - ?::interval)
			ORDER BY next_attempt_at,created_at FOR UPDATE SKIP LOCKED LIMIT 1
		)
		UPDATE invoice_reminders r SET attempts=r.attempts+1,locked_at=now(),updated_at=now()
		FROM candidate c WHERE r.id=c.id
		RETURNING r.id::text AS id,r.recipient_email,r.subject,r.message,r.attempts,r.max_attempts`, lease.String()).Scan(&delivery)
	if result.Error != nil {
		return nil, fmt.Errorf("claim reminder delivery: %w", result.Error)
	}
	if result.RowsAffected == 0 || delivery.ID == "" {
		return nil, nil
	}
	return &delivery, nil
}

func (r *PostgresRepository) MarkDelivered(ctx context.Context, deliveryID, providerMessageID string) error {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE invoice_reminders SET status='sent',sent_at=now(),provider_message_id=?,locked_at=NULL,last_error=NULL,updated_at=now()
		WHERE id=? AND status='queued' AND locked_at IS NOT NULL`, providerMessageID, deliveryID)
	if result.Error != nil {
		return fmt.Errorf("complete reminder delivery: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return apperrors.ErrInvalidState
	}
	return nil
}

func (r *PostgresRepository) MarkDeliveryFailed(ctx context.Context, delivery Delivery, cause error) error {
	message := cause.Error()
	if len(message) > 500 {
		message = message[:500]
	}
	backoff := time.Duration(delivery.Attempts*delivery.Attempts) * time.Minute
	result := r.db.WithContext(ctx).Exec(`
		UPDATE invoice_reminders SET
			status=CASE WHEN attempts >= max_attempts THEN 'failed' ELSE 'queued' END,
			failed_at=CASE WHEN attempts >= max_attempts THEN now() ELSE NULL END,
			last_error=?,locked_at=NULL,next_attempt_at=now() + ?::interval,updated_at=now()
		WHERE id=? AND status='queued' AND locked_at IS NOT NULL`, message, backoff.String(), delivery.ID)
	if result.Error != nil {
		return fmt.Errorf("fail reminder delivery: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return apperrors.ErrInvalidState
	}
	return nil
}

func (r *PostgresRepository) RecordDeliveryEvent(ctx context.Context, input DeliveryEventInput, occurredAt time.Time) error {
	payload := input.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var reminderID string
		if err := tx.Raw(`SELECT id::text FROM invoice_reminders WHERE provider_message_id=?`, input.ProviderMessageID).Scan(&reminderID).Error; err != nil {
			return fmt.Errorf("find delivered reminder: %w", err)
		}
		if reminderID == "" {
			return apperrors.ErrNotFound
		}
		result := tx.Exec(`
			INSERT INTO reminder_delivery_events (organization_id,reminder_id,provider_event_id,provider_message_id,event_type,occurred_at,payload)
			SELECT organization_id,id,?,?,?,?,?::jsonb FROM invoice_reminders WHERE id=?
			ON CONFLICT (provider_event_id) DO NOTHING`, input.EventID, input.ProviderMessageID, input.EventType, occurredAt, string(payload), reminderID)
		if result.Error != nil {
			return fmt.Errorf("record delivery event: %w", result.Error)
		}
		return nil
	})
}

func (r *PostgresRepository) GenerateDue(ctx context.Context, organizationID string) (int64, error) {
	return r.generate(ctx, "AND i.organization_id = ?", organizationID)
}

func (r *PostgresRepository) GenerateAllDue(ctx context.Context) (int64, error) {
	return r.generate(ctx, "", nil)
}

func (r *PostgresRepository) generate(ctx context.Context, organizationFilter string, argument any) (int64, error) {
	query := fmt.Sprintf(`
		INSERT INTO invoice_reminders (organization_id,invoice_id,client_id,contact_id,sequence,status,scheduled_for,recipient_email,subject,message)
		SELECT i.organization_id,i.id,i.client_id,cc.id,1,'queued',CURRENT_DATE,cc.email,
			'Payment reminder: ' || i.invoice_number,
			'Hello ' || cc.name || ', invoice ' || i.invoice_number || ' is overdue. Please review the outstanding balance.'
		FROM invoices i
		JOIN invoice_balances b ON b.invoice_id=i.id AND b.organization_id=i.organization_id
		JOIN clients c ON c.id=i.client_id JOIN client_contacts cc ON cc.client_id=c.id AND cc.is_primary
		WHERE c.reminders_enabled AND i.status IN ('open','partial','overdue')
			AND i.due_on + c.reminder_delay_days <= CURRENT_DATE
			AND b.outstanding_minor > 0
			%s ON CONFLICT (invoice_id,sequence) DO NOTHING`, organizationFilter)
	db := r.db.WithContext(ctx)
	var result *gorm.DB
	if organizationFilter == "" {
		result = db.Exec(query)
	} else {
		result = db.Exec(query, argument)
	}
	if result.Error != nil {
		return 0, fmt.Errorf("generate overdue reminders: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *PostgresRepository) Cancel(ctx context.Context, organizationID, reminderID string, actor Actor) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var invoiceID string
		if err := tx.Raw(`SELECT invoice_id::text FROM invoice_reminders WHERE id=? AND organization_id=? AND status='queued' AND locked_at IS NULL FOR UPDATE`, reminderID, organizationID).Scan(&invoiceID).Error; err != nil {
			return err
		}
		if invoiceID == "" {
			return apperrors.ErrNotFound
		}
		if err := tx.Exec(`UPDATE invoice_reminders SET status='cancelled',cancelled_at=now(),cancelled_by=? WHERE id=?`, actor.UserID, reminderID).Error; err != nil {
			return fmt.Errorf("cancel reminder: %w", err)
		}
		return audit.Write(tx, organizationID, actor.Email, "reminder.cancelled", "invoice", invoiceID, map[string]any{"reminderId": reminderID})
	})
}
