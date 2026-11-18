package adjustments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/portfolio/ledgerops/backend/internal/platform/apperrors"
	"gorm.io/gorm"
)

type PostgresRepository struct{ db *gorm.DB }

func NewPostgresRepository(db *gorm.DB) *PostgresRepository { return &PostgresRepository{db: db} }

func (r *PostgresRepository) List(ctx context.Context, organizationID, invoiceID string) ([]Adjustment, error) {
	var exists bool
	if err := r.db.WithContext(ctx).Raw(`SELECT EXISTS(SELECT 1 FROM invoices WHERE id = ? AND organization_id = ?)`, invoiceID, organizationID).Scan(&exists).Error; err != nil {
		return nil, fmt.Errorf("check adjustment invoice: %w", err)
	}
	if !exists {
		return nil, apperrors.ErrNotFound
	}
	rows := make([]Adjustment, 0)
	if err := r.db.WithContext(ctx).Raw(`
		SELECT a.id::text AS id, a.invoice_id::text AS invoice_id, COALESCE(a.payment_id::text, '') AS payment_id,
			a.kind, a.amount_minor, i.currency_code, a.reason, a.status,
			a.requested_by::text AS requested_by_id, requester.email AS requested_by,
			COALESCE(reviewer.email, '') AS reviewed_by, COALESCE(a.review_note, '') AS review_note,
			to_char(a.requested_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS requested_at,
			COALESCE(to_char(a.reviewed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '') AS reviewed_at
		FROM invoice_adjustments a
		JOIN invoices i ON i.id = a.invoice_id
		JOIN users requester ON requester.id = a.requested_by
		LEFT JOIN users reviewer ON reviewer.id = a.reviewed_by
		WHERE a.organization_id = ? AND a.invoice_id = ?
		ORDER BY a.requested_at DESC`, organizationID, invoiceID).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("list invoice adjustments: %w", err)
	}
	return rows, nil
}

func (r *PostgresRepository) Create(ctx context.Context, organizationID, invoiceID string, actor Actor, input CreateInput) (Adjustment, error) {
	var adjustmentID string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		invoice, err := lockInvoice(tx, organizationID, invoiceID)
		if err != nil {
			return err
		}
		if invoice.Status == "draft" || invoice.Status == "void" {
			return apperrors.ErrInvalidState
		}

		amount := input.AmountMinor
		if input.Kind == KindCreditNote {
			totals, err := loadTotals(tx, invoiceID)
			if err != nil {
				return err
			}
			var pendingCredits int64
			if err := tx.Raw(`SELECT COALESCE(SUM(amount_minor), 0)::bigint FROM invoice_adjustments WHERE invoice_id = ? AND kind = 'credit_note' AND status = 'pending'`, invoiceID).Scan(&pendingCredits).Error; err != nil {
				return fmt.Errorf("sum pending credits: %w", err)
			}
			available := invoice.AmountMinor - totals.ApprovedCredits - totals.NetPaid - pendingCredits
			if amount > available {
				return apperrors.ErrInvalidState
			}
		} else {
			var payment struct {
				AmountMinor int64
				Currency    string
			}
			if err := tx.Raw(`SELECT amount_minor, currency_code AS currency FROM payments WHERE id = ? AND invoice_id = ? AND organization_id = ?`, input.PaymentID, invoiceID, organizationID).Scan(&payment).Error; err != nil {
				return fmt.Errorf("load reversal payment: %w", err)
			}
			if payment.AmountMinor == 0 || payment.Currency != invoice.CurrencyCode {
				return apperrors.ErrInvalidReference
			}
			amount = payment.AmountMinor
		}

		row := struct{ ID string }{}
		result := tx.Raw(`
			INSERT INTO invoice_adjustments (organization_id, invoice_id, payment_id, kind, amount_minor, reason, requested_by)
			VALUES (?, ?, NULLIF(?, '')::uuid, ?, ?, ?, ?)
			RETURNING id::text AS id`, organizationID, invoiceID, input.PaymentID, input.Kind, amount, input.Reason, actor.UserID).Scan(&row)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
				return apperrors.ErrInvalidState
			}
			return fmt.Errorf("insert invoice adjustment: %w", result.Error)
		}
		adjustmentID = row.ID
		return writeAudit(tx, organizationID, actor.Email, "adjustment.requested", invoiceID, map[string]any{
			"adjustmentId": adjustmentID, "kind": input.Kind, "amountMinor": amount, "reason": input.Reason,
		})
	})
	if err != nil {
		return Adjustment{}, fmt.Errorf("create adjustment transaction: %w", err)
	}
	return r.get(ctx, organizationID, adjustmentID)
}

func (r *PostgresRepository) Decide(ctx context.Context, organizationID, adjustmentID string, actor Actor, input DecisionInput) (DecisionResult, error) {
	var invoiceID string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var lookup struct{ InvoiceID string }
		if err := tx.Raw(`SELECT invoice_id::text AS invoice_id FROM invoice_adjustments WHERE id = ? AND organization_id = ?`, adjustmentID, organizationID).Scan(&lookup).Error; err != nil {
			return fmt.Errorf("find adjustment invoice: %w", err)
		}
		if lookup.InvoiceID == "" {
			return apperrors.ErrNotFound
		}
		invoiceID = lookup.InvoiceID
		invoice, err := lockInvoice(tx, organizationID, invoiceID)
		if err != nil {
			return err
		}

		var pending struct {
			ID            string
			InvoiceID     string
			Kind          Kind
			AmountMinor   int64
			RequestedByID string
			Status        string
		}
		if err := tx.Raw(`
			SELECT id::text AS id, invoice_id::text AS invoice_id, kind, amount_minor,
				requested_by::text AS requested_by_id, status
			FROM invoice_adjustments
			WHERE id = ? AND organization_id = ? FOR UPDATE`, adjustmentID, organizationID).Scan(&pending).Error; err != nil {
			return fmt.Errorf("load adjustment decision: %w", err)
		}
		if pending.ID == "" {
			return apperrors.ErrNotFound
		}
		if pending.Status != "pending" || pending.RequestedByID == actor.UserID {
			return apperrors.ErrInvalidState
		}
		if pending.InvoiceID != invoiceID {
			return apperrors.ErrInvalidState
		}
		if input.Decision == "approved" {
			totals, err := loadTotals(tx, invoiceID)
			if err != nil {
				return err
			}
			if pending.Kind == KindCreditNote && pending.AmountMinor > invoice.AmountMinor-totals.ApprovedCredits-totals.NetPaid {
				return apperrors.ErrInvalidState
			}
			if pending.Kind == KindPaymentReversal && pending.AmountMinor > totals.NetPaid {
				return apperrors.ErrInvalidState
			}
		}

		if err := tx.Exec(`
			UPDATE invoice_adjustments SET status = ?, reviewed_by = ?, review_note = ?, reviewed_at = now()
			WHERE id = ?`, input.Decision, actor.UserID, input.Note, adjustmentID).Error; err != nil {
			return fmt.Errorf("finalize invoice adjustment: %w", err)
		}
		if input.Decision == "approved" {
			if err := deriveInvoiceStatus(tx, invoice); err != nil {
				return err
			}
		}
		return writeAudit(tx, organizationID, actor.Email, "adjustment."+input.Decision, invoiceID, map[string]any{
			"adjustmentId": adjustmentID, "kind": pending.Kind, "amountMinor": pending.AmountMinor, "note": input.Note,
		})
	})
	if err != nil {
		return DecisionResult{}, fmt.Errorf("decide adjustment transaction: %w", err)
	}
	adjustment, err := r.get(ctx, organizationID, adjustmentID)
	if err != nil {
		return DecisionResult{}, err
	}
	invoice, err := r.getInvoice(ctx, organizationID, invoiceID)
	if err != nil {
		return DecisionResult{}, err
	}
	return DecisionResult{Adjustment: adjustment, Invoice: invoice}, nil
}

type lockedInvoice struct {
	ID           string
	Status       string
	CurrencyCode string
	AmountMinor  int64
	DueOn        string
}

func lockInvoice(tx *gorm.DB, organizationID, invoiceID string) (lockedInvoice, error) {
	var invoice lockedInvoice
	if err := tx.Raw(`SELECT id::text AS id, status, currency_code, amount_minor, to_char(due_on, 'YYYY-MM-DD') AS due_on FROM invoices WHERE id = ? AND organization_id = ? FOR UPDATE`, invoiceID, organizationID).Scan(&invoice).Error; err != nil {
		return invoice, fmt.Errorf("lock adjustment invoice: %w", err)
	}
	if invoice.ID == "" {
		return invoice, apperrors.ErrNotFound
	}
	return invoice, nil
}

type totals struct {
	ApprovedCredits int64
	NetPaid         int64
}

func loadTotals(tx *gorm.DB, invoiceID string) (totals, error) {
	var result totals
	if err := tx.Raw(`SELECT credit_minor AS approved_credits, paid_minor AS net_paid FROM invoice_balances WHERE invoice_id = ?`, invoiceID).Scan(&result).Error; err != nil {
		return result, fmt.Errorf("load invoice adjustment totals: %w", err)
	}
	return result, nil
}

func deriveInvoiceStatus(tx *gorm.DB, invoice lockedInvoice) error {
	totals, err := loadTotals(tx, invoice.ID)
	if err != nil {
		return err
	}
	effectiveAmount := invoice.AmountMinor - totals.ApprovedCredits
	status := "open"
	if totals.NetPaid == effectiveAmount {
		status = "paid"
	} else if totals.NetPaid > 0 {
		status = "partial"
	} else if invoice.DueOn < time.Now().Format(time.DateOnly) {
		status = "overdue"
	}
	if err := tx.Table("invoices").Where("id = ?", invoice.ID).Updates(map[string]any{"status": status, "updated_at": gorm.Expr("now()")}).Error; err != nil {
		return fmt.Errorf("derive adjusted invoice status: %w", err)
	}
	return nil
}

func (r *PostgresRepository) get(ctx context.Context, organizationID, adjustmentID string) (Adjustment, error) {
	var result Adjustment
	if err := r.db.WithContext(ctx).Raw(`
		SELECT a.id::text AS id, a.invoice_id::text AS invoice_id, COALESCE(a.payment_id::text, '') AS payment_id,
			a.kind, a.amount_minor, i.currency_code, a.reason, a.status,
			a.requested_by::text AS requested_by_id, requester.email AS requested_by,
			COALESCE(reviewer.email, '') AS reviewed_by, COALESCE(a.review_note, '') AS review_note,
			to_char(a.requested_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS requested_at,
			COALESCE(to_char(a.reviewed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '') AS reviewed_at
		FROM invoice_adjustments a JOIN invoices i ON i.id = a.invoice_id
		JOIN users requester ON requester.id = a.requested_by LEFT JOIN users reviewer ON reviewer.id = a.reviewed_by
		WHERE a.id = ? AND a.organization_id = ?`, adjustmentID, organizationID).Scan(&result).Error; err != nil {
		return result, fmt.Errorf("get invoice adjustment: %w", err)
	}
	if result.ID == "" {
		return result, apperrors.ErrNotFound
	}
	return result, nil
}

func (r *PostgresRepository) getInvoice(ctx context.Context, organizationID, invoiceID string) (InvoiceSnapshot, error) {
	var result InvoiceSnapshot
	if err := r.db.WithContext(ctx).Raw(`
		SELECT i.id::text AS id, i.client_id::text AS client_id, i.invoice_number, c.name AS client_name,
			i.status, i.currency_code, i.amount_minor, b.paid_minor, b.outstanding_minor,
			to_char(i.issued_on, 'YYYY-MM-DD') AS issued_on, to_char(i.due_on, 'YYYY-MM-DD') AS due_on, i.source,
			to_char(i.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at,
			to_char(i.updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS updated_at
		FROM invoices i
		JOIN clients c ON c.id = i.client_id
		JOIN invoice_balances b ON b.invoice_id = i.id AND b.organization_id = i.organization_id
		WHERE i.id = ? AND i.organization_id = ?`, invoiceID, organizationID).Scan(&result).Error; err != nil {
		return result, fmt.Errorf("get adjusted invoice: %w", err)
	}
	if result.ID == "" {
		return result, apperrors.ErrNotFound
	}
	return result, nil
}

func writeAudit(tx *gorm.DB, organizationID, actor, action, invoiceID string, metadata map[string]any) error {
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("encode adjustment audit metadata: %w", err)
	}
	if err := tx.Exec(`INSERT INTO audit_events (organization_id, actor, action, entity_type, entity_id, metadata) VALUES (?, ?, ?, 'invoice', ?, ?::jsonb)`, organizationID, actor, action, invoiceID, string(encoded)).Error; err != nil {
		return fmt.Errorf("write adjustment audit event: %w", err)
	}
	return nil
}
