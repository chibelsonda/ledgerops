package invoices

import (
	"context"
	"errors"
	"fmt"

	"github.com/portfolio/ledgerops/backend/internal/financial"
	"github.com/portfolio/ledgerops/backend/internal/platform/apperrors"
	"github.com/portfolio/ledgerops/backend/internal/platform/audit"
	"gorm.io/gorm"
)

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) List(ctx context.Context, organizationID string, params ListParams) (Page, error) {
	filters := r.db.WithContext(ctx).
		Table("invoices i").
		Joins("JOIN clients c ON c.id = i.client_id").
		Joins("JOIN invoice_balances b ON b.invoice_id = i.id AND b.organization_id = i.organization_id").
		Where("i.organization_id = ?", organizationID)
	if params.Query != "" {
		like := "%" + params.Query + "%"
		filters = filters.Where("i.invoice_number ILIKE ? OR c.name ILIKE ?", like, like)
	}
	if params.Status != "" {
		filters = filters.Where("i.status = ?", params.Status)
	}

	var total int64
	if err := filters.Session(&gorm.Session{}).Distinct("i.id").Count(&total).Error; err != nil {
		return Page{}, fmt.Errorf("count invoices: %w", err)
	}

	items := make([]Invoice, 0)
	err := filters.
		Select(`i.id::text AS id, i.client_id::text AS client_id, i.invoice_number, c.name AS client_name,
			i.status, i.currency_code, i.amount_minor, b.paid_minor, b.outstanding_minor,
			to_char(i.issued_on, 'YYYY-MM-DD') AS issued_on,
			to_char(i.due_on, 'YYYY-MM-DD') AS due_on,
			i.source,
			to_char(i.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at,
			to_char(i.updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS updated_at`).
		Order("i.issued_on DESC, i.id DESC").
		Limit(params.PageSize).
		Offset((params.Page - 1) * params.PageSize).
		Scan(&items).Error
	if err != nil {
		return Page{}, fmt.Errorf("list invoices: %w", err)
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(params.PageSize) - 1) / int64(params.PageSize))
	}
	return Page{
		Items:      items,
		Pagination: Pagination{Page: params.Page, PageSize: params.PageSize, TotalItems: total, TotalPages: totalPages},
	}, nil
}

func (r *PostgresRepository) ListClients(ctx context.Context, organizationID string) ([]ClientOption, error) {
	clients := make([]ClientOption, 0)
	if err := r.db.WithContext(ctx).Raw(`
		SELECT id::text AS id, name
		FROM clients
		WHERE organization_id = ?
		ORDER BY name`, organizationID).Scan(&clients).Error; err != nil {
		return nil, fmt.Errorf("list invoice clients: %w", err)
	}
	return clients, nil
}

func (r *PostgresRepository) Create(ctx context.Context, organizationID, actor string, input MutationInput) (Invoice, error) {
	var invoiceID string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureClient(tx, organizationID, input.ClientID); err != nil {
			return err
		}
		row := struct{ ID string }{}
		result := tx.Raw(`
			INSERT INTO invoices (
				organization_id, client_id, invoice_number, status, currency_code,
				amount_minor, issued_on, due_on, source
			) VALUES (?, ?, ?, ?, ?, ?, ?::date, ?::date, 'manual')
			RETURNING id::text AS id`,
			organizationID, input.ClientID, input.InvoiceNumber, input.Status, input.CurrencyCode,
			input.AmountMinor, input.IssuedOn, input.DueOn,
		).Scan(&row)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
				return apperrors.ErrConflict
			}
			return fmt.Errorf("insert invoice: %w", result.Error)
		}
		invoiceID = row.ID
		metadata := map[string]any{
			"invoiceNumber": input.InvoiceNumber,
			"status":        input.Status,
			"amountMinor":   input.AmountMinor,
			"currencyCode":  input.CurrencyCode,
		}
		return insertAuditEvent(tx, organizationID, actor, "invoice.created", invoiceID, metadata)
	})
	if err != nil {
		return Invoice{}, fmt.Errorf("create invoice transaction: %w", err)
	}
	return r.get(ctx, organizationID, invoiceID)
}

func (r *PostgresRepository) Update(ctx context.Context, organizationID, invoiceID, actor string, input MutationInput) (Invoice, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureClient(tx, organizationID, input.ClientID); err != nil {
			return err
		}

		var before struct {
			ClientID      string
			InvoiceNumber string
			Status        string
			CurrencyCode  string
			AmountMinor   int64
			IssuedOn      string
			DueOn         string
			Source        string
		}
		if err := tx.Raw(`
			SELECT client_id::text AS client_id, invoice_number, status, currency_code, amount_minor,source,
				to_char(issued_on, 'YYYY-MM-DD') AS issued_on,
				to_char(due_on, 'YYYY-MM-DD') AS due_on
			FROM invoices
			WHERE id = ? AND organization_id = ?
			FOR UPDATE`, invoiceID, organizationID).Scan(&before).Error; err != nil {
			return fmt.Errorf("load invoice for update: %w", err)
		}
		if before.InvoiceNumber == "" {
			return apperrors.ErrNotFound
		}
		if before.Source == "project_billing" {
			return apperrors.ErrInvalidState
		}
		paidTotal, err := sumNetPayments(tx, invoiceID)
		if err != nil {
			return err
		}
		credits, err := sumApprovedCredits(tx, invoiceID)
		if err != nil {
			return err
		}
		status, allowed := financial.ReconcileInvoiceMutation(input.Status, input.AmountMinor, credits, paidTotal)
		if !allowed {
			return apperrors.ErrInvalidState
		}
		input.Status = status

		result := tx.Table("invoices").
			Where("id = ? AND organization_id = ?", invoiceID, organizationID).
			Updates(map[string]any{
				"client_id":      input.ClientID,
				"invoice_number": input.InvoiceNumber,
				"status":         input.Status,
				"currency_code":  input.CurrencyCode,
				"amount_minor":   input.AmountMinor,
				"issued_on":      input.IssuedOn,
				"due_on":         input.DueOn,
				"updated_at":     gorm.Expr("now()"),
			})
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
				return apperrors.ErrConflict
			}
			return fmt.Errorf("update invoice: %w", result.Error)
		}

		changes := changedFields(before, input)
		metadata := map[string]any{"note": input.AuditNote, "changes": changes}
		return insertAuditEvent(tx, organizationID, actor, "invoice.updated", invoiceID, metadata)
	})
	if err != nil {
		return Invoice{}, fmt.Errorf("update invoice transaction: %w", err)
	}
	return r.get(ctx, organizationID, invoiceID)
}

func (r *PostgresRepository) ListAuditEvents(ctx context.Context, organizationID, invoiceID string) ([]AuditEvent, error) {
	var exists bool
	if err := r.db.WithContext(ctx).Raw(
		"SELECT EXISTS(SELECT 1 FROM invoices WHERE id = ? AND organization_id = ?)",
		invoiceID, organizationID,
	).Scan(&exists).Error; err != nil {
		return nil, fmt.Errorf("check invoice for audit events: %w", err)
	}
	if !exists {
		return nil, apperrors.ErrNotFound
	}

	type auditRow struct {
		ID        string
		Actor     string
		Action    string
		Metadata  string
		CreatedAt string
	}
	rows := make([]auditRow, 0)
	if err := r.db.WithContext(ctx).Raw(`
		SELECT id::text AS id, actor, action, metadata::text AS metadata,
			to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at
		FROM audit_events
		WHERE organization_id = ? AND entity_type = 'invoice' AND entity_id = ?
		ORDER BY created_at DESC`, organizationID, invoiceID).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("list invoice audit events: %w", err)
	}

	events := make([]AuditEvent, 0, len(rows))
	for _, row := range rows {
		metadata, err := audit.DecodeMetadata(row.Metadata)
		if err != nil {
			return nil, fmt.Errorf("decode invoice audit metadata: %w", err)
		}
		events = append(events, AuditEvent{ID: row.ID, Actor: row.Actor, Action: row.Action, Metadata: metadata, CreatedAt: row.CreatedAt})
	}
	return events, nil
}

func (r *PostgresRepository) ListPayments(ctx context.Context, organizationID, invoiceID string) ([]Payment, error) {
	var exists bool
	if err := r.db.WithContext(ctx).Raw(
		"SELECT EXISTS(SELECT 1 FROM invoices WHERE id = ? AND organization_id = ?)",
		invoiceID, organizationID,
	).Scan(&exists).Error; err != nil {
		return nil, fmt.Errorf("check invoice for payments: %w", err)
	}
	if !exists {
		return nil, apperrors.ErrNotFound
	}

	payments := make([]Payment, 0)
	if err := r.db.WithContext(ctx).Raw(`
		SELECT p.id::text AS id, p.invoice_id::text AS invoice_id, p.currency_code, p.amount_minor,
			to_char(p.received_on, 'YYYY-MM-DD') AS received_on,
			COALESCE(p.reference, '') AS reference, p.source,
			COALESCE(u.email, CASE WHEN p.source = 'manual' THEN 'Unknown user' ELSE 'Imported from ' || p.source END) AS recorded_by,
			EXISTS(SELECT 1 FROM invoice_adjustments a WHERE a.payment_id = p.id AND a.kind = 'payment_reversal' AND a.status = 'approved') AS reversed,
			to_char(p.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at
		FROM payments p
		LEFT JOIN users u ON u.id = p.created_by
		WHERE p.organization_id = ? AND p.invoice_id = ?
		ORDER BY p.received_on DESC, p.created_at DESC`, organizationID, invoiceID).Scan(&payments).Error; err != nil {
		return nil, fmt.Errorf("list invoice payments: %w", err)
	}
	return payments, nil
}

func (r *PostgresRepository) RecordPayment(ctx context.Context, organizationID, invoiceID string, actor Actor, input PaymentInput) (PaymentResult, error) {
	var paymentID string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var invoice struct {
			ID           string
			Status       string
			CurrencyCode string
			AmountMinor  int64
		}
		if err := tx.Raw(`
			SELECT id::text AS id, status, currency_code, amount_minor
			FROM invoices
			WHERE id = ? AND organization_id = ?
			FOR UPDATE`, invoiceID, organizationID).Scan(&invoice).Error; err != nil {
			return fmt.Errorf("load invoice for payment: %w", err)
		}
		if invoice.ID == "" {
			return apperrors.ErrNotFound
		}
		if invoice.Status == "draft" || invoice.Status == "void" || invoice.Status == "paid" {
			return apperrors.ErrInvalidState
		}

		paidTotal, err := sumNetPayments(tx, invoiceID)
		if err != nil {
			return err
		}
		credits, err := sumApprovedCredits(tx, invoiceID)
		if err != nil {
			return err
		}
		effectiveAmount := invoice.AmountMinor - credits
		newStatus, allowed := financial.StatusAfterPayment(effectiveAmount, paidTotal, input.AmountMinor)
		if !allowed {
			return apperrors.ErrInvalidState
		}

		row := struct{ ID string }{}
		if err := tx.Raw(`
			INSERT INTO payments (
				organization_id, invoice_id, currency_code, amount_minor, received_on,
				source, reference, created_by
			) VALUES (?, ?, ?, ?, ?::date, 'manual', NULLIF(?, ''), ?)
			RETURNING id::text AS id`,
			organizationID, invoiceID, invoice.CurrencyCode, input.AmountMinor, input.ReceivedOn,
			input.Reference, actor.UserID,
		).Scan(&row).Error; err != nil {
			return fmt.Errorf("insert invoice payment: %w", err)
		}
		paymentID = row.ID

		if err := tx.Table("invoices").Where("id = ? AND organization_id = ?", invoiceID, organizationID).Updates(map[string]any{
			"status":     newStatus,
			"updated_at": gorm.Expr("now()"),
		}).Error; err != nil {
			return fmt.Errorf("derive invoice status after payment: %w", err)
		}

		metadata := map[string]any{
			"paymentId":   paymentID,
			"amountMinor": input.AmountMinor,
			"receivedOn":  input.ReceivedOn,
			"reference":   input.Reference,
			"note":        input.AuditNote,
			"status":      newStatus,
		}
		return insertAuditEvent(tx, organizationID, actor.Email, "payment.recorded", invoiceID, metadata)
	})
	if err != nil {
		return PaymentResult{}, fmt.Errorf("record payment transaction: %w", err)
	}

	payment, err := r.getPayment(ctx, organizationID, paymentID)
	if err != nil {
		return PaymentResult{}, err
	}
	invoice, err := r.get(ctx, organizationID, invoiceID)
	if err != nil {
		return PaymentResult{}, err
	}
	return PaymentResult{Payment: payment, Invoice: invoice}, nil
}

func (r *PostgresRepository) get(ctx context.Context, organizationID, invoiceID string) (Invoice, error) {
	var invoice Invoice
	err := r.db.WithContext(ctx).Raw(`
		SELECT i.id::text AS id, i.client_id::text AS client_id, i.invoice_number, c.name AS client_name,
			i.status, i.currency_code, i.amount_minor, b.paid_minor, b.outstanding_minor,
			to_char(i.issued_on, 'YYYY-MM-DD') AS issued_on,
			to_char(i.due_on, 'YYYY-MM-DD') AS due_on,
			i.source,
			to_char(i.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at,
			to_char(i.updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS updated_at
		FROM invoices i
		JOIN clients c ON c.id = i.client_id
		JOIN invoice_balances b ON b.invoice_id = i.id AND b.organization_id = i.organization_id
		WHERE i.id = ? AND i.organization_id = ?`, invoiceID, organizationID).Scan(&invoice).Error
	if err != nil {
		return Invoice{}, fmt.Errorf("get invoice: %w", err)
	}
	if invoice.ID == "" {
		return Invoice{}, apperrors.ErrNotFound
	}
	return invoice, nil
}

func (r *PostgresRepository) getPayment(ctx context.Context, organizationID, paymentID string) (Payment, error) {
	var payment Payment
	if err := r.db.WithContext(ctx).Raw(`
		SELECT p.id::text AS id, p.invoice_id::text AS invoice_id, p.currency_code, p.amount_minor,
			to_char(p.received_on, 'YYYY-MM-DD') AS received_on,
			COALESCE(p.reference, '') AS reference, p.source,
			COALESCE(u.email, 'Unknown user') AS recorded_by,
			EXISTS(SELECT 1 FROM invoice_adjustments a WHERE a.payment_id = p.id AND a.kind = 'payment_reversal' AND a.status = 'approved') AS reversed,
			to_char(p.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at
		FROM payments p
		LEFT JOIN users u ON u.id = p.created_by
		WHERE p.id = ? AND p.organization_id = ?`, paymentID, organizationID).Scan(&payment).Error; err != nil {
		return Payment{}, fmt.Errorf("get invoice payment: %w", err)
	}
	if payment.ID == "" {
		return Payment{}, apperrors.ErrNotFound
	}
	return payment, nil
}

func sumNetPayments(tx *gorm.DB, invoiceID string) (int64, error) {
	var total int64
	if err := tx.Raw(`SELECT paid_minor FROM invoice_balances WHERE invoice_id = ?`, invoiceID).Scan(&total).Error; err != nil {
		return 0, fmt.Errorf("sum net invoice payments: %w", err)
	}
	return total, nil
}

func sumApprovedCredits(tx *gorm.DB, invoiceID string) (int64, error) {
	var total int64
	if err := tx.Raw("SELECT credit_minor FROM invoice_balances WHERE invoice_id = ?", invoiceID).Scan(&total).Error; err != nil {
		return 0, fmt.Errorf("sum approved invoice credits: %w", err)
	}
	return total, nil
}

func ensureClient(tx *gorm.DB, organizationID, clientID string) error {
	var exists bool
	if err := tx.Raw(
		"SELECT EXISTS(SELECT 1 FROM clients WHERE id = ? AND organization_id = ?)",
		clientID, organizationID,
	).Scan(&exists).Error; err != nil {
		return fmt.Errorf("check invoice client: %w", err)
	}
	if !exists {
		return apperrors.ErrInvalidReference
	}
	return nil
}

func insertAuditEvent(tx *gorm.DB, organizationID, actor, action, entityID string, metadata map[string]any) error {
	return audit.Write(tx, organizationID, actor, action, "invoice", entityID, metadata)
}

func changedFields(before struct {
	ClientID      string
	InvoiceNumber string
	Status        string
	CurrencyCode  string
	AmountMinor   int64
	IssuedOn      string
	DueOn         string
	Source        string
}, after MutationInput) map[string]map[string]any {
	changes := make(map[string]map[string]any)
	add := func(field string, oldValue, newValue any) {
		if oldValue != newValue {
			changes[field] = map[string]any{"from": oldValue, "to": newValue}
		}
	}
	add("clientId", before.ClientID, after.ClientID)
	add("invoiceNumber", before.InvoiceNumber, after.InvoiceNumber)
	add("status", before.Status, after.Status)
	add("currencyCode", before.CurrencyCode, after.CurrencyCode)
	add("amountMinor", before.AmountMinor, after.AmountMinor)
	add("issuedOn", before.IssuedOn, after.IssuedOn)
	add("dueOn", before.DueOn, after.DueOn)
	return changes
}
