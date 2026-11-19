package quickbooks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"gorm.io/gorm"
)

type OutboxJob struct {
	ID             string
	OrganizationID string
	AggregateType  string
	AggregateID    string
	Operation      string
	Attempts       int
}

type OutboxRepository interface {
	FindConnection(context.Context, string) (*Connection, error)
	UpdateTokens(context.Context, string, []byte, []byte, time.Time, time.Time, string) error
	ClaimOutbox(context.Context) (*OutboxJob, error)
	OutboundPayload(context.Context, OutboxJob) (map[string]any, error)
	CompleteOutbox(context.Context, OutboxJob, string, string) error
	FailOutbox(context.Context, OutboxJob, error, bool, time.Duration) error
}

type OutboundAccountingClient interface {
	RefreshTokens(context.Context, string) (Tokens, error)
	WriteEntity(context.Context, string, string, string, map[string]any) (RemoteEntity, error)
}

func (r *PostgresRepository) ClaimOutbox(ctx context.Context) (*OutboxJob, error) {
	var job OutboxJob
	result := r.db.WithContext(ctx).Raw(`WITH candidate AS (
		SELECT id FROM quickbooks_outbox
		WHERE (status = 'queued' AND next_attempt_at <= now())
			OR (status = 'running' AND locked_at < now() - interval '10 minutes')
		ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1
	) UPDATE quickbooks_outbox o SET status = 'running', locked_at = now(), attempts = attempts + 1, updated_at = now()
	FROM candidate c WHERE o.id = c.id RETURNING o.id::text AS id, o.organization_id::text AS organization_id,
		o.aggregate_type, o.aggregate_id::text AS aggregate_id, o.operation, o.attempts`).Scan(&job)
	if result.Error != nil {
		return nil, fmt.Errorf("claim QuickBooks outbox: %w", result.Error)
	}
	if result.RowsAffected == 0 || job.ID == "" {
		return nil, nil
	}
	return &job, nil
}

func (r *PostgresRepository) OutboundPayload(ctx context.Context, job OutboxJob) (map[string]any, error) {
	payload := make(map[string]any)
	switch job.AggregateType {
	case "client":
		var row struct{ Name, Email, ExternalID, SyncToken string }
		result := r.db.WithContext(ctx).Raw(`SELECT c.name, COALESCE(c.email, cc.email, '') AS email,
			COALESCE(c.external_id, '') AS external_id, COALESCE(q.sync_token, '') AS sync_token
			FROM clients c LEFT JOIN client_contacts cc ON cc.client_id = c.id AND cc.is_primary
			LEFT JOIN quickbooks_entities q ON q.organization_id = c.organization_id AND q.entity_type = 'Customer'
				AND q.external_id = split_part(c.external_id, ':', 3)
			WHERE c.id = ? AND c.organization_id = ?`, job.AggregateID, job.OrganizationID).Scan(&row)
		if result.Error != nil || result.RowsAffected == 0 {
			return nil, errors.Join(ErrConnectionMissing, result.Error)
		}
		payload["DisplayName"] = row.Name
		if row.Email != "" {
			payload["PrimaryEmailAddr"] = map[string]string{"Address": row.Email}
		}
		addSparseIdentity(payload, row.ExternalID, row.SyncToken)
	case "invoice":
		var row struct {
			Number, Currency, IssuedOn, DueOn, ExternalID, SyncToken, CustomerExternalID string
			AmountMinor                                                                  int64
		}
		result := r.db.WithContext(ctx).Raw(`SELECT i.invoice_number AS number, i.currency_code AS currency,
			to_char(i.issued_on, 'YYYY-MM-DD') AS issued_on, to_char(i.due_on, 'YYYY-MM-DD') AS due_on,
			COALESCE(i.external_id, '') AS external_id, COALESCE(q.sync_token, '') AS sync_token,
			COALESCE(c.external_id, '') AS customer_external_id, i.amount_minor
			FROM invoices i JOIN clients c ON c.id = i.client_id
			LEFT JOIN quickbooks_entities q ON q.organization_id = i.organization_id AND q.entity_type = 'Invoice'
				AND q.external_id = split_part(i.external_id, ':', 3)
			WHERE i.id = ? AND i.organization_id = ?`, job.AggregateID, job.OrganizationID).Scan(&row)
		if result.Error != nil || result.RowsAffected == 0 {
			return nil, errors.Join(ErrConnectionMissing, result.Error)
		}
		customerID := externalIDSuffix(row.CustomerExternalID)
		if !strings.HasPrefix(row.CustomerExternalID, "quickbooks:customer:") {
			return nil, fmt.Errorf("client must synchronize before its invoice")
		}
		payload["CustomerRef"] = map[string]string{"value": customerID}
		payload["DocNumber"], payload["TxnDate"], payload["DueDate"] = row.Number, row.IssuedOn, row.DueOn
		payload["CurrencyRef"] = map[string]string{"value": row.Currency}
		payload["Line"] = []any{map[string]any{"Amount": float64(row.AmountMinor) / 100,
			"DetailType": "SalesItemLineDetail", "Description": "LedgerOps invoice " + row.Number,
			"SalesItemLineDetail": map[string]any{}}}
		addSparseIdentity(payload, row.ExternalID, row.SyncToken)
	case "payment":
		var row struct {
			Currency, ReceivedOn, Reference, InvoiceExternalID, CustomerExternalID string
			AmountMinor                                                            int64
		}
		result := r.db.WithContext(ctx).Raw(`SELECT p.currency_code AS currency, to_char(p.received_on, 'YYYY-MM-DD') AS received_on,
			COALESCE(p.reference, '') AS reference, COALESCE(i.external_id, '') AS invoice_external_id,
			COALESCE(c.external_id, '') AS customer_external_id, p.amount_minor
			FROM payments p JOIN invoices i ON i.id = p.invoice_id JOIN clients c ON c.id = i.client_id
			WHERE p.id = ? AND p.organization_id = ?`, job.AggregateID, job.OrganizationID).Scan(&row)
		if result.Error != nil || result.RowsAffected == 0 {
			return nil, errors.Join(ErrConnectionMissing, result.Error)
		}
		if !strings.HasPrefix(row.InvoiceExternalID, "quickbooks:invoice:") {
			return nil, fmt.Errorf("invoice must synchronize before its payment")
		}
		if !strings.HasPrefix(row.CustomerExternalID, "quickbooks:customer:") {
			return nil, fmt.Errorf("client must synchronize before its payment")
		}
		payload["TotalAmt"] = float64(row.AmountMinor) / 100
		payload["TxnDate"], payload["PaymentRefNum"] = row.ReceivedOn, row.Reference
		payload["CurrencyRef"] = map[string]string{"value": row.Currency}
		payload["CustomerRef"] = map[string]string{"value": externalIDSuffix(row.CustomerExternalID)}
		payload["Line"] = []any{map[string]any{"Amount": float64(row.AmountMinor) / 100,
			"LinkedTxn": []any{map[string]string{"TxnId": externalIDSuffix(row.InvoiceExternalID), "TxnType": "Invoice"}}}}
	default:
		return nil, &PermanentError{Err: fmt.Errorf("unsupported QuickBooks outbox aggregate %q", job.AggregateType)}
	}
	return payload, nil
}

func addSparseIdentity(payload map[string]any, externalID, syncToken string) {
	if externalID == "" {
		return
	}
	payload["Id"], payload["SyncToken"], payload["sparse"] = externalIDSuffix(externalID), syncToken, true
}

func (r *PostgresRepository) CompleteOutbox(ctx context.Context, job OutboxJob, externalID, syncToken string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`UPDATE quickbooks_outbox SET status = 'succeeded', provider_external_id = ?,
			locked_at = NULL, completed_at = now(), updated_at = now() WHERE id = ?`, externalID, job.ID).Error; err != nil {
			return err
		}
		entityType := map[string]string{"client": EntityCustomer, "invoice": EntityInvoice, "payment": EntityPayment}[job.AggregateType]
		if entityType == "" {
			return fmt.Errorf("unsupported QuickBooks outbox aggregate %q", job.AggregateType)
		}
		if err := tx.Exec(`INSERT INTO quickbooks_entities (
			organization_id, entity_type, external_id, sync_token, payload, source_updated_at
		) VALUES (?, ?, ?, NULLIF(?, ''), '{}'::jsonb, now())
		ON CONFLICT (organization_id, entity_type, external_id) DO UPDATE SET
			sync_token = EXCLUDED.sync_token, deleted_at = NULL, updated_at = now()`,
			job.OrganizationID, entityType, externalID, syncToken).Error; err != nil {
			return fmt.Errorf("store outbound QuickBooks %s identity: %w", entityType, err)
		}
		prefixed := externalID
		switch job.AggregateType {
		case "client":
			prefixed = externalIDForOutbound("customer", externalID)
			return tx.Table("clients").Where("id = ? AND organization_id = ?", job.AggregateID, job.OrganizationID).Update("external_id", prefixed).Error
		case "invoice":
			prefixed = externalIDForOutbound("invoice", externalID)
			return tx.Table("invoices").Where("id = ? AND organization_id = ?", job.AggregateID, job.OrganizationID).Update("external_id", prefixed).Error
		case "payment":
			prefixed = externalIDForOutbound("payment", externalID) + ":0:0"
			return tx.Table("payments").Where("id = ? AND organization_id = ?", job.AggregateID, job.OrganizationID).Update("external_id", prefixed).Error
		}
		return nil
	})
}

func externalIDForOutbound(entityType, id string) string {
	return "quickbooks:" + entityType + ":" + id
}

func (r *PostgresRepository) FailOutbox(ctx context.Context, job OutboxJob, cause error, retryable bool, retryAfter time.Duration) error {
	if retryAfter <= 0 {
		retryAfter = time.Duration(job.Attempts*job.Attempts) * time.Minute
	}
	return r.db.WithContext(ctx).Exec(`UPDATE quickbooks_outbox SET status = CASE WHEN ? AND attempts < max_attempts THEN 'queued' ELSE 'failed' END,
		last_error = ?, locked_at = NULL, next_attempt_at = now() + ?::interval, updated_at = now() WHERE id = ?`,
		retryable, cause.Error(), retryAfter.String(), job.ID).Error
}

type OutboxWorker struct {
	repository OutboxRepository
	client     OutboundAccountingClient
	cipher     Cipher
	logger     *slog.Logger
	interval   time.Duration
	now        func() time.Time
}

func NewOutboxWorker(repository OutboxRepository, client OutboundAccountingClient, cipher Cipher, logger *slog.Logger, interval time.Duration) *OutboxWorker {
	return &OutboxWorker{repository: repository, client: client, cipher: cipher, logger: logger, interval: interval, now: time.Now}
}

func (w *OutboxWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		if !w.runOnce(ctx) {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}
}

func (w *OutboxWorker) runOnce(ctx context.Context) bool {
	job, err := w.repository.ClaimOutbox(ctx)
	if err != nil {
		w.logger.Error("claim QuickBooks outbox", "error", err)
		return false
	}
	if job == nil {
		return false
	}
	err = w.process(ctx, *job)
	if err != nil {
		retryable, delay := retryDetails(err)
		if failErr := w.repository.FailOutbox(ctx, *job, err, retryable, delay); failErr != nil {
			w.logger.Error("fail QuickBooks outbox", "error", failErr)
		}
		w.logger.Error("write QuickBooks entity", "aggregate", job.AggregateType, "aggregate_id", job.AggregateID, "error", err)
	}
	return true
}

func (w *OutboxWorker) process(ctx context.Context, job OutboxJob) error {
	connection, err := w.repository.FindConnection(ctx, job.OrganizationID)
	if err != nil {
		return err
	}
	if connection == nil {
		return &PermanentError{Err: ErrConnectionMissing}
	}
	access, err := w.cipher.Decrypt(connection.AccessToken)
	if err != nil {
		return &PermanentError{Err: err}
	}
	refresh, err := w.cipher.Decrypt(connection.RefreshToken)
	if err != nil {
		return &PermanentError{Err: err}
	}
	if !connection.AccessTokenExpiresAt.After(w.now().Add(tokenRefreshLeeway)) {
		tokens, refreshErr := w.client.RefreshTokens(ctx, refresh)
		if refreshErr != nil {
			return refreshErr
		}
		accessCipher, encryptErr := w.cipher.Encrypt(tokens.AccessToken)
		if encryptErr != nil {
			return &PermanentError{Err: encryptErr}
		}
		refreshCipher, encryptErr := w.cipher.Encrypt(tokens.RefreshToken)
		if encryptErr != nil {
			return &PermanentError{Err: encryptErr}
		}
		if err := w.repository.UpdateTokens(ctx, job.OrganizationID, accessCipher, refreshCipher, w.now().Add(tokens.AccessTokenExpiresIn), w.now().Add(tokens.RefreshTokenExpiresIn), tokens.Scope); err != nil {
			return err
		}
		access = tokens.AccessToken
	}
	payload, err := w.repository.OutboundPayload(ctx, job)
	if err != nil {
		return err
	}
	entityType := map[string]string{"client": EntityCustomer, "invoice": EntityInvoice, "payment": EntityPayment}[job.AggregateType]
	entity, err := w.client.WriteEntity(ctx, access, connection.RealmID, entityType, payload)
	if err != nil {
		return err
	}
	return w.repository.CompleteOutbox(ctx, job, entity.ExternalID, entity.SyncToken)
}

func retryDetails(err error) (bool, time.Duration) {
	var retryable interface {
		Retryable() bool
		RetryAfter() time.Duration
	}
	if errors.As(err, &retryable) {
		return retryable.Retryable(), retryable.RetryAfter()
	}
	return true, 0
}
