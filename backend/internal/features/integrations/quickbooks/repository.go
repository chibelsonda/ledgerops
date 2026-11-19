package quickbooks

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

var ErrInvalidState = errors.New("invalid or expired OAuth state")

type Repository interface {
	FindConnection(context.Context, string) (*Connection, error)
	SaveState(context.Context, string, []byte, time.Time) error
	ConsumeState(context.Context, []byte, time.Time) (string, error)
	UpsertConnection(context.Context, Connection) error
	DeleteConnection(context.Context, string) error
	Operations(context.Context, string) (OperationalStatus, error)
}

func (r *PostgresRepository) Operations(ctx context.Context, organizationID string) (OperationalStatus, error) {
	var status OperationalStatus
	if err := r.db.WithContext(ctx).Raw(`SELECT
		COALESCE((SELECT enabled FROM quickbooks_sync_schedules WHERE organization_id = ?), false) AS schedule_enabled,
		(SELECT next_run_at FROM quickbooks_sync_schedules WHERE organization_id = ?) AS next_scheduled_at,
		(SELECT MAX(finished_at) FROM sync_runs WHERE organization_id = ? AND provider = 'quickbooks-sandbox' AND status = 'succeeded') AS last_successful_sync,
		(SELECT COUNT(*) FROM quickbooks_webhook_events WHERE organization_id = ? AND processed_at IS NULL) AS pending_webhooks,
		(SELECT COUNT(*) FROM quickbooks_outbox WHERE organization_id = ? AND status IN ('queued','running')) AS queued_writes,
		(SELECT COUNT(*) FROM quickbooks_outbox WHERE organization_id = ? AND status = 'failed') AS failed_writes`,
		organizationID, organizationID, organizationID, organizationID, organizationID, organizationID).Scan(&status).Error; err != nil {
		return OperationalStatus{}, fmt.Errorf("load QuickBooks operational status: %w", err)
	}
	return status, nil
}

type SyncRepository interface {
	FindConnection(context.Context, string) (*Connection, error)
	UpdateTokens(context.Context, string, []byte, []byte, time.Time, time.Time, string) error
	UpsertCustomers(context.Context, string, []CustomerRecord) error
	UpsertInvoices(context.Context, string, []InvoiceRecord) error
	InsertPayments(context.Context, string, []PaymentRecord) error
	UpsertExpenses(context.Context, string, []ExpenseRecord) error
	UpsertRemoteEntities(context.Context, string, []RemoteEntity) error
	ReconcileFullSnapshot(context.Context, string, string, time.Time) ([]RemoteEntity, error)
	ReconcileDeletedEntities(context.Context, string, []RemoteEntity) error
	Cursor(context.Context, string, string) (*time.Time, error)
	SaveCursor(context.Context, string, string, time.Time, bool) error
	UpdateEntityProgress(context.Context, string, string, string, int, string) error
	MarkWebhookEventsProcessed(context.Context, string, time.Time) error
}

func (r *PostgresRepository) ReconcileFullSnapshot(ctx context.Context, organizationID, entityType string, startedAt time.Time) ([]RemoteEntity, error) {
	missing := make([]RemoteEntity, 0)
	if err := r.db.WithContext(ctx).Raw(`UPDATE quickbooks_entities
		SET deleted_at = ?, updated_at = now()
		WHERE organization_id = ? AND entity_type = ? AND deleted_at IS NULL AND updated_at < ?
		RETURNING entity_type, external_id, deleted_at`, startedAt, organizationID, entityType, startedAt).Scan(&missing).Error; err != nil {
		return nil, fmt.Errorf("reconcile full QuickBooks %s snapshot: %w", entityType, err)
	}
	return missing, nil
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) FindConnection(ctx context.Context, organizationID string) (*Connection, error) {
	var connection Connection
	result := r.db.WithContext(ctx).Raw(`
		SELECT organization_id::text AS organization_id, realm_id, company_name,
			access_token_ciphertext AS access_token, refresh_token_ciphertext AS refresh_token,
			access_token_expires_at, refresh_token_expires_at, scope, connected_at
		FROM quickbooks_connections WHERE organization_id = ?`, organizationID).Scan(&connection)
	if result.Error != nil {
		return nil, fmt.Errorf("find QuickBooks connection: %w", result.Error)
	}
	if result.RowsAffected == 0 || connection.OrganizationID == "" {
		return nil, nil
	}
	return &connection, nil
}

func (r *PostgresRepository) SaveState(ctx context.Context, organizationID string, stateHash []byte, expiresAt time.Time) error {
	if err := r.db.WithContext(ctx).Exec(`
		INSERT INTO quickbooks_oauth_states (organization_id, state_hash, expires_at)
		VALUES (?, ?, ?)`, organizationID, stateHash, expiresAt).Error; err != nil {
		return fmt.Errorf("save QuickBooks OAuth state: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ConsumeState(ctx context.Context, stateHash []byte, now time.Time) (string, error) {
	var organizationID string
	result := r.db.WithContext(ctx).Raw(`
		UPDATE quickbooks_oauth_states
		SET consumed_at = ?
		WHERE state_hash = ? AND consumed_at IS NULL AND expires_at > ?
		RETURNING organization_id::text`, now, stateHash, now).Scan(&organizationID)
	if result.Error != nil {
		return "", fmt.Errorf("consume QuickBooks OAuth state: %w", result.Error)
	}
	if result.RowsAffected == 0 || organizationID == "" {
		return "", ErrInvalidState
	}
	return organizationID, nil
}

func (r *PostgresRepository) UpsertConnection(ctx context.Context, connection Connection) error {
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
		INSERT INTO quickbooks_connections (
			organization_id, realm_id, company_name, access_token_ciphertext,
			refresh_token_ciphertext, access_token_expires_at,
			refresh_token_expires_at, scope
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (organization_id) DO UPDATE SET
			realm_id = EXCLUDED.realm_id,
			company_name = EXCLUDED.company_name,
			access_token_ciphertext = EXCLUDED.access_token_ciphertext,
			refresh_token_ciphertext = EXCLUDED.refresh_token_ciphertext,
			access_token_expires_at = EXCLUDED.access_token_expires_at,
			refresh_token_expires_at = EXCLUDED.refresh_token_expires_at,
			scope = EXCLUDED.scope,
			connected_at = now(),
			updated_at = now()`,
			connection.OrganizationID, connection.RealmID, connection.CompanyName,
			connection.AccessToken, connection.RefreshToken, connection.AccessTokenExpiresAt,
			connection.RefreshTokenExpiresAt, connection.Scope,
		).Error; err != nil {
			return fmt.Errorf("save QuickBooks connection: %w", err)
		}
		if err := tx.Exec(`INSERT INTO quickbooks_sync_schedules (organization_id, next_run_at)
		VALUES (?, now()) ON CONFLICT (organization_id) DO UPDATE SET enabled = true,
		next_run_at = LEAST(quickbooks_sync_schedules.next_run_at, now()), updated_at = now()`, connection.OrganizationID).Error; err != nil {
			return fmt.Errorf("enable QuickBooks sync schedule: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (r *PostgresRepository) DeleteConnection(ctx context.Context, organizationID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM quickbooks_sync_schedules WHERE organization_id = ?", organizationID).Error; err != nil {
			return fmt.Errorf("delete QuickBooks schedule: %w", err)
		}
		if err := tx.Exec("DELETE FROM quickbooks_outbox WHERE organization_id = ? AND status IN ('queued', 'running')", organizationID).Error; err != nil {
			return fmt.Errorf("delete pending QuickBooks writes: %w", err)
		}
		if err := tx.Exec("DELETE FROM quickbooks_connections WHERE organization_id = ?", organizationID).Error; err != nil {
			return fmt.Errorf("delete QuickBooks connection: %w", err)
		}
		return nil
	})
}

func (r *PostgresRepository) UpdateTokens(
	ctx context.Context,
	organizationID string,
	accessToken, refreshToken []byte,
	accessExpiresAt, refreshExpiresAt time.Time,
	scope string,
) error {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE quickbooks_connections SET access_token_ciphertext = ?, refresh_token_ciphertext = ?,
			access_token_expires_at = ?, refresh_token_expires_at = ?,
			scope = CASE WHEN ? = '' THEN scope ELSE ? END, updated_at = now()
		WHERE organization_id = ?`, accessToken, refreshToken, accessExpiresAt, refreshExpiresAt, scope, scope, organizationID)
	if result.Error != nil {
		return fmt.Errorf("update QuickBooks tokens: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("update QuickBooks tokens: connection no longer exists")
	}
	return nil
}

func (r *PostgresRepository) UpsertCustomers(ctx context.Context, organizationID string, records []CustomerRecord) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, record := range records {
			var clientID string
			if err := tx.Raw(`
				INSERT INTO clients (organization_id, external_id, name, email)
				VALUES (?, ?, ?, NULLIF(?, ''))
				ON CONFLICT (organization_id, external_id) DO UPDATE SET
					name = EXCLUDED.name, email = EXCLUDED.email, source_deleted_at = NULL, updated_at = now()
				RETURNING id::text`, organizationID, record.ExternalID, record.Name, record.Email,
			).Scan(&clientID).Error; err != nil {
				return fmt.Errorf("upsert QuickBooks customer %s: %w", record.ExternalID, err)
			}
			if record.Email == "" {
				continue
			}
			if err := tx.Exec(`
				INSERT INTO client_contacts (organization_id, client_id, name, email, role_title, is_primary)
				VALUES (?, ?, ?, ?, 'Billing contact',
					NOT EXISTS (SELECT 1 FROM client_contacts WHERE client_id = ? AND is_primary))
				ON CONFLICT (client_id, email) DO UPDATE SET name = EXCLUDED.name, updated_at = now()`,
				organizationID, clientID, record.Name+" billing", record.Email, clientID,
			).Error; err != nil {
				return fmt.Errorf("upsert QuickBooks customer contact %s: %w", record.ExternalID, err)
			}
		}
		return nil
	})
}

func (r *PostgresRepository) UpsertInvoices(ctx context.Context, organizationID string, records []InvoiceRecord) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, record := range records {
			var clientID string
			result := tx.Raw(`SELECT id::text FROM clients WHERE organization_id = ? AND external_id = ?`,
				organizationID, record.CustomerExternalID).Scan(&clientID)
			if result.Error != nil {
				return fmt.Errorf("find QuickBooks invoice customer %s: %w", record.CustomerExternalID, result.Error)
			}
			if result.RowsAffected == 0 || clientID == "" {
				return fmt.Errorf("QuickBooks invoice %s references an unknown customer", record.ExternalID)
			}

			invoiceNumber := record.Number
			var numberInUse bool
			if err := tx.Raw(`
				SELECT EXISTS(
					SELECT 1 FROM invoices
					WHERE organization_id = ? AND invoice_number = ?
						AND external_id IS DISTINCT FROM ?
				)`, organizationID, invoiceNumber, record.ExternalID).Scan(&numberInUse).Error; err != nil {
				return fmt.Errorf("check QuickBooks invoice number %s: %w", record.Number, err)
			}
			if numberInUse {
				invoiceNumber = record.Number + "-QBO-" + externalIDSuffix(record.ExternalID)
			}

			update := tx.Exec(`UPDATE invoices SET client_id = ?, invoice_number = ?, status = ?, currency_code = ?,
				amount_minor = ?, issued_on = ?, due_on = ?, source_updated_at = ?, source_deleted_at = NULL, updated_at = now()
				WHERE organization_id = ? AND external_id = ?`, clientID, invoiceNumber, record.Status,
				record.CurrencyCode, record.AmountMinor, record.IssuedOn, record.DueOn, record.SourceUpdatedAt,
				organizationID, record.ExternalID)
			if update.Error != nil {
				return fmt.Errorf("update QuickBooks invoice %s: %w", record.ExternalID, update.Error)
			}
			if update.RowsAffected > 0 {
				continue
			}

			if err := tx.Exec(`
				INSERT INTO invoices (
					organization_id, client_id, external_id, invoice_number, status,
					currency_code, amount_minor, issued_on, due_on, source, source_updated_at
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'quickbooks', ?)
				ON CONFLICT (organization_id, source, external_id) DO UPDATE SET
					client_id = EXCLUDED.client_id, invoice_number = EXCLUDED.invoice_number,
					status = EXCLUDED.status, currency_code = EXCLUDED.currency_code,
					amount_minor = EXCLUDED.amount_minor, issued_on = EXCLUDED.issued_on,
					due_on = EXCLUDED.due_on, source_updated_at = EXCLUDED.source_updated_at,
					source_deleted_at = NULL, updated_at = now()`,
				organizationID, clientID, record.ExternalID, invoiceNumber, record.Status,
				record.CurrencyCode, record.AmountMinor, record.IssuedOn, record.DueOn, record.SourceUpdatedAt,
			).Error; err != nil {
				return fmt.Errorf("upsert QuickBooks invoice %s: %w", record.ExternalID, err)
			}
		}
		return nil
	})
}

func (r *PostgresRepository) InsertPayments(ctx context.Context, organizationID string, records []PaymentRecord) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, record := range records {
			var invoiceID string
			result := tx.Raw(`
				SELECT id::text FROM invoices
				WHERE organization_id = ? AND external_id = ?`,
				organizationID, record.InvoiceExternalID).Scan(&invoiceID)
			if result.Error != nil {
				return fmt.Errorf("find QuickBooks payment invoice %s: %w", record.InvoiceExternalID, result.Error)
			}
			if result.RowsAffected == 0 || invoiceID == "" {
				return fmt.Errorf("QuickBooks payment %s references an unknown invoice", record.ExternalID)
			}
			var exists bool
			if err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM payments WHERE organization_id = ? AND external_id = ?)`,
				organizationID, record.ExternalID).Scan(&exists).Error; err != nil {
				return fmt.Errorf("find QuickBooks payment %s: %w", record.ExternalID, err)
			}
			if exists {
				if err := tx.Table("payments").Where("organization_id = ? AND external_id = ? AND source_deleted_at IS NOT NULL", organizationID, record.ExternalID).Update("source_deleted_at", nil).Error; err != nil {
					return fmt.Errorf("restore QuickBooks payment %s: %w", record.ExternalID, err)
				}
				continue
			}
			if err := tx.Exec(`
				INSERT INTO payments (
					organization_id, invoice_id, external_id, currency_code,
					amount_minor, received_on, source, reference
				) VALUES (?, ?, ?, ?, ?, ?, 'quickbooks', NULLIF(?, ''))
				ON CONFLICT (organization_id, source, external_id) DO UPDATE SET
					source_deleted_at = NULL
				WHERE payments.source_deleted_at IS NOT NULL`,
				organizationID, invoiceID, record.ExternalID, record.CurrencyCode,
				record.AmountMinor, record.ReceivedOn, record.Reference,
			).Error; err != nil {
				return fmt.Errorf("insert QuickBooks payment %s: %w", record.ExternalID, err)
			}
		}
		return nil
	})
}

func (r *PostgresRepository) UpsertExpenses(ctx context.Context, organizationID string, records []ExpenseRecord) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, record := range records {
			if err := tx.Exec(`
				INSERT INTO expenses (
					organization_id, external_id, category, vendor_name,
					currency_code, amount_minor, incurred_on, source
				) VALUES (?, ?, ?, ?, ?, ?, ?, 'quickbooks')
				ON CONFLICT (organization_id, source, external_id) DO UPDATE SET
					category = EXCLUDED.category, vendor_name = EXCLUDED.vendor_name,
					currency_code = EXCLUDED.currency_code, amount_minor = EXCLUDED.amount_minor,
					incurred_on = EXCLUDED.incurred_on, source_deleted_at = NULL`,
				organizationID, record.ExternalID, record.Category, record.VendorName,
				record.CurrencyCode, record.AmountMinor, record.IncurredOn,
			).Error; err != nil {
				return fmt.Errorf("upsert QuickBooks expense %s: %w", record.ExternalID, err)
			}
		}
		return nil
	})
}

func externalIDSuffix(value string) string {
	for index := len(value) - 1; index >= 0; index-- {
		if value[index] == ':' {
			return value[index+1:]
		}
	}
	return value
}

func (r *PostgresRepository) UpsertRemoteEntities(ctx context.Context, organizationID string, records []RemoteEntity) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, record := range records {
			if err := tx.Exec(`
				INSERT INTO quickbooks_entities (
					organization_id, entity_type, external_id, sync_token, display_name,
					document_number, status, currency_code, amount_minor, balance_minor,
					transaction_date, source_updated_at, payload, deleted_at
				) VALUES (?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''),
					NULLIF(?, ''), ?, ?, ?, ?, ?::jsonb, ?)
				ON CONFLICT (organization_id, entity_type, external_id) DO UPDATE SET
					sync_token = EXCLUDED.sync_token, display_name = EXCLUDED.display_name,
					document_number = EXCLUDED.document_number, status = EXCLUDED.status,
					currency_code = EXCLUDED.currency_code, amount_minor = EXCLUDED.amount_minor,
					balance_minor = EXCLUDED.balance_minor, transaction_date = EXCLUDED.transaction_date,
					source_updated_at = EXCLUDED.source_updated_at, payload = EXCLUDED.payload,
					deleted_at = EXCLUDED.deleted_at, updated_at = now()`,
				organizationID, record.EntityType, record.ExternalID, record.SyncToken, record.DisplayName,
				record.DocumentNumber, record.Status, record.CurrencyCode, record.AmountMinor,
				record.BalanceMinor, record.TransactionDate, record.SourceUpdatedAt, string(record.Payload), record.DeletedAt,
			).Error; err != nil {
				return fmt.Errorf("upsert QuickBooks %s %s: %w", record.EntityType, record.ExternalID, err)
			}
		}
		return nil
	})
}

func (r *PostgresRepository) ReconcileDeletedEntities(ctx context.Context, organizationID string, records []RemoteEntity) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, record := range records {
			if record.DeletedAt == nil && !(record.EntityType == EntityInvoice && strings.EqualFold(record.Status, "Voided")) {
				continue
			}
			deletedAt := record.DeletedAt
			if deletedAt == nil {
				now := time.Now().UTC()
				deletedAt = &now
			}
			externalID := externalID(strings.ToLower(record.EntityType), record.ExternalID)
			var result *gorm.DB
			switch record.EntityType {
			case EntityCustomer:
				result = tx.Table("clients").Where("organization_id = ? AND external_id = ?", organizationID, externalID).
					Update("source_deleted_at", deletedAt)
			case EntityInvoice:
				result = tx.Table("invoices").Where("organization_id = ? AND external_id = ?", organizationID, externalID).
					Updates(map[string]any{"source_deleted_at": deletedAt, "status": "void", "updated_at": gorm.Expr("now()")})
			case EntityPayment:
				result = tx.Table("payments").Where("organization_id = ? AND external_id LIKE ?", organizationID, externalID+":%").
					Update("source_deleted_at", deletedAt)
			case EntityPurchase:
				result = tx.Table("expenses").Where("organization_id = ? AND source = 'quickbooks' AND external_id = ?", organizationID, externalID).
					Update("source_deleted_at", deletedAt)
			default:
				continue
			}
			if result.Error != nil {
				return fmt.Errorf("reconcile deleted QuickBooks %s %s: %w", record.EntityType, record.ExternalID, result.Error)
			}
		}
		return nil
	})
}

func (r *PostgresRepository) Cursor(ctx context.Context, organizationID, entityType string) (*time.Time, error) {
	var cursor time.Time
	result := r.db.WithContext(ctx).Raw(`
		SELECT changed_since FROM quickbooks_sync_cursors
		WHERE organization_id = ? AND entity_type = ?`, organizationID, entityType).Scan(&cursor)
	if result.Error != nil {
		return nil, fmt.Errorf("get QuickBooks %s cursor: %w", entityType, result.Error)
	}
	if result.RowsAffected == 0 || cursor.IsZero() {
		return nil, nil
	}
	return &cursor, nil
}

func (r *PostgresRepository) SaveCursor(ctx context.Context, organizationID, entityType string, cursor time.Time, full bool) error {
	if err := r.db.WithContext(ctx).Exec(`
		INSERT INTO quickbooks_sync_cursors (
			organization_id, entity_type, changed_since, last_full_sync_at, last_success_at
		) VALUES (?, ?, ?::timestamptz, CASE WHEN ?::boolean THEN ?::timestamptz ELSE NULL END, now())
		ON CONFLICT (organization_id, entity_type) DO UPDATE SET
			changed_since = EXCLUDED.changed_since,
			last_full_sync_at = CASE WHEN ?::boolean THEN EXCLUDED.last_full_sync_at ELSE quickbooks_sync_cursors.last_full_sync_at END,
			last_success_at = now(), updated_at = now()`,
		organizationID, entityType, cursor, full, cursor, full,
	).Error; err != nil {
		return fmt.Errorf("save QuickBooks %s cursor: %w", entityType, err)
	}
	return nil
}

func (r *PostgresRepository) UpdateEntityProgress(
	ctx context.Context,
	syncRunID, entityType, status string,
	recordsProcessed int,
	errorMessage string,
) error {
	if err := r.db.WithContext(ctx).Exec(`
		INSERT INTO sync_run_entities (
			sync_run_id, entity_type, status, records_processed, error_message, started_at, finished_at
		) VALUES (?, ?, ?, ?, NULLIF(?, ''),
			CASE WHEN ? IN ('running', 'succeeded', 'failed') THEN now() ELSE NULL END,
			CASE WHEN ? IN ('succeeded', 'failed') THEN now() ELSE NULL END)
		ON CONFLICT (sync_run_id, entity_type) DO UPDATE SET
			status = EXCLUDED.status, records_processed = EXCLUDED.records_processed,
			error_message = EXCLUDED.error_message,
			started_at = COALESCE(sync_run_entities.started_at, EXCLUDED.started_at),
			finished_at = EXCLUDED.finished_at`,
		syncRunID, entityType, status, recordsProcessed, errorMessage, status, status,
	).Error; err != nil {
		return fmt.Errorf("update QuickBooks %s progress: %w", entityType, err)
	}
	return nil
}

func (r *PostgresRepository) MarkWebhookEventsProcessed(ctx context.Context, organizationID string, through time.Time) error {
	if err := r.db.WithContext(ctx).Exec(`
		UPDATE quickbooks_webhook_events SET processed_at = now()
		WHERE organization_id = ? AND processed_at IS NULL AND occurred_at <= ?`, organizationID, through).Error; err != nil {
		return fmt.Errorf("mark QuickBooks webhook events processed: %w", err)
	}
	return nil
}

func (r *PostgresRepository) StoreWebhookEvents(ctx context.Context, events []WebhookEvent) ([]string, error) {
	organizations := make(map[string]struct{})
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, event := range events {
			var organizationID string
			result := tx.Raw(`SELECT organization_id::text FROM quickbooks_connections WHERE realm_id = ?`, event.RealmID).Scan(&organizationID)
			if result.Error != nil {
				return fmt.Errorf("find QuickBooks webhook organization: %w", result.Error)
			}
			if result.RowsAffected == 0 || organizationID == "" {
				continue
			}
			insert := tx.Exec(`
				INSERT INTO quickbooks_webhook_events (
					event_id, organization_id, realm_id, entity_type, entity_id,
					operation, occurred_at, payload
				) VALUES (?, ?, ?, ?, NULLIF(?, ''), ?, ?, ?::jsonb)
				ON CONFLICT (event_id) DO NOTHING`, event.EventID, organizationID, event.RealmID,
				event.EntityType, event.EntityID, event.Operation, event.OccurredAt, string(event.Payload))
			if insert.Error != nil {
				return fmt.Errorf("store QuickBooks webhook event %s: %w", event.EventID, insert.Error)
			}
			if insert.RowsAffected > 0 {
				organizations[organizationID] = struct{}{}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	organizationIDs := make([]string, 0, len(organizations))
	for organizationID := range organizations {
		organizationIDs = append(organizationIDs, organizationID)
	}
	return organizationIDs, nil
}
