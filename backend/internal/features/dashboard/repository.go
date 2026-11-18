package dashboard

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/portfolio/ledgerops/backend/internal/platform/apperrors"
	"gorm.io/gorm"
)

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Get(ctx context.Context, organizationID string) (View, error) {
	var result View
	query := r.db.WithContext(ctx).
		Table("organizations").
		Select("id::text AS id, name, base_currency").
		Where("id = ?", organizationID).
		Take(&result.Organization)
	if query.Error != nil {
		if errors.Is(query.Error, gorm.ErrRecordNotFound) {
			return result, apperrors.ErrNotFound
		}
		return result, fmt.Errorf("load organization: %w", query.Error)
	}
	if err := r.loadMetrics(ctx, organizationID, &result.Metrics); err != nil {
		return result, err
	}
	var err error
	if result.Trend, err = r.loadTrend(ctx, organizationID); err != nil {
		return result, err
	}
	if result.Budget, err = r.loadBudget(ctx, organizationID); err != nil {
		return result, err
	}
	if result.Aging, err = r.loadAging(ctx, organizationID); err != nil {
		return result, err
	}
	return result, nil
}

func (r *PostgresRepository) loadMetrics(ctx context.Context, organizationID string, metrics *Metrics) error {
	err := r.db.WithContext(ctx).Raw(`
		WITH paid AS (
			SELECT p.invoice_id, (SUM(p.amount_minor) - COALESCE(MAX(reversed.total), 0))::bigint AS paid_minor
			FROM payments p
			LEFT JOIN (SELECT invoice_id, SUM(amount_minor)::bigint AS total FROM invoice_adjustments WHERE organization_id = @organization_id AND kind = 'payment_reversal' AND status = 'approved' GROUP BY invoice_id) reversed ON reversed.invoice_id = p.invoice_id
			WHERE p.organization_id = @organization_id GROUP BY p.invoice_id
		), credits AS (
			SELECT invoice_id, SUM(amount_minor)::bigint AS credit_minor FROM invoice_adjustments
			WHERE organization_id = @organization_id AND kind = 'credit_note' AND status = 'approved' GROUP BY invoice_id
		)
		SELECT
			((SELECT COALESCE(SUM(amount_minor), 0)::bigint FROM payments WHERE organization_id = @organization_id)
			 - (SELECT COALESCE(SUM(amount_minor), 0)::bigint FROM invoice_adjustments WHERE organization_id = @organization_id AND kind = 'payment_reversal' AND status = 'approved'))
			 - (SELECT COALESCE(SUM(amount_minor), 0)::bigint FROM expenses WHERE organization_id = @organization_id) AS cash_position_minor,
			(SELECT COALESCE(SUM(i.amount_minor - COALESCE(c.credit_minor, 0)), 0)::bigint FROM invoices i LEFT JOIN credits c ON c.invoice_id = i.id WHERE i.organization_id = @organization_id AND i.issued_on >= date_trunc('month', CURRENT_DATE)) AS revenue_mtd_minor,
			(SELECT COALESCE(SUM(GREATEST(i.amount_minor - COALESCE(c.credit_minor, 0) - COALESCE(p.paid_minor, 0), 0)), 0)::bigint
			 FROM invoices i LEFT JOIN paid p ON p.invoice_id = i.id LEFT JOIN credits c ON c.invoice_id = i.id
			 WHERE i.organization_id = @organization_id AND i.status IN ('open', 'partial', 'overdue')) AS outstanding_ar_minor,
			(SELECT COALESCE(SUM(amount_minor), 0)::bigint FROM expenses WHERE organization_id = @organization_id AND incurred_on >= date_trunc('month', CURRENT_DATE)) AS expenses_mtd_minor,
			(SELECT COUNT(*)::int FROM reconciliation_items WHERE organization_id = @organization_id AND status = 'open') AS open_review_count`,
		sql.Named("organization_id", organizationID),
	).Scan(metrics).Error
	if err != nil {
		return fmt.Errorf("load dashboard metrics: %w", err)
	}
	return nil
}

func (r *PostgresRepository) loadTrend(ctx context.Context, organizationID string) ([]TrendPoint, error) {
	points := make([]TrendPoint, 0, 6)
	err := r.db.WithContext(ctx).Raw(`
		WITH months AS (
			SELECT generate_series(date_trunc('month', CURRENT_DATE) - interval '5 months', date_trunc('month', CURRENT_DATE), interval '1 month')::date AS month_start
		), credits AS (
			SELECT invoice_id, SUM(amount_minor)::bigint AS total FROM invoice_adjustments
			WHERE organization_id = @organization_id AND kind = 'credit_note' AND status = 'approved' GROUP BY invoice_id
		), revenue AS (
			SELECT date_trunc('month', i.issued_on)::date AS month_start, SUM(i.amount_minor - COALESCE(c.total, 0))::bigint AS total
			FROM invoices i LEFT JOIN credits c ON c.invoice_id = i.id WHERE i.organization_id = @organization_id GROUP BY 1
		), spend AS (
			SELECT date_trunc('month', incurred_on)::date AS month_start, SUM(amount_minor)::bigint AS total
			FROM expenses WHERE organization_id = @organization_id GROUP BY 1
		)
		SELECT to_char(m.month_start, 'Mon') AS month,
			COALESCE(r.total, 0)::bigint AS revenue_minor,
			COALESCE(e.total, 0)::bigint AS expenses_minor
		FROM months m LEFT JOIN revenue r USING (month_start) LEFT JOIN spend e USING (month_start)
		ORDER BY m.month_start`, sql.Named("organization_id", organizationID)).Scan(&points).Error
	if err != nil {
		return nil, fmt.Errorf("load revenue trend: %w", err)
	}
	return points, nil
}

func (r *PostgresRepository) loadBudget(ctx context.Context, organizationID string) ([]BudgetCategory, error) {
	categories := make([]BudgetCategory, 0)
	err := r.db.WithContext(ctx).Raw(`
		WITH actual AS (
			SELECT category, SUM(amount_minor)::bigint AS total FROM expenses
			WHERE organization_id = @organization_id AND incurred_on >= date_trunc('month', CURRENT_DATE) GROUP BY category
		)
		SELECT b.category, b.amount_minor AS budget_minor,
			COALESCE(a.total, 0)::bigint AS actual_minor,
			(b.amount_minor - COALESCE(a.total, 0))::bigint AS variance_minor
		FROM budgets b LEFT JOIN actual a USING (category)
		WHERE b.organization_id = @organization_id AND b.month_start = date_trunc('month', CURRENT_DATE)::date
		ORDER BY b.amount_minor DESC`, sql.Named("organization_id", organizationID)).Scan(&categories).Error
	if err != nil {
		return nil, fmt.Errorf("load budget comparison: %w", err)
	}
	return categories, nil
}

func (r *PostgresRepository) loadAging(ctx context.Context, organizationID string) ([]AgingBucket, error) {
	buckets := make([]AgingBucket, 0, 4)
	err := r.db.WithContext(ctx).Raw(`
		WITH paid AS (
			SELECT p.invoice_id, (SUM(p.amount_minor) - COALESCE(MAX(reversed.total), 0))::bigint AS paid_minor
			FROM payments p LEFT JOIN (SELECT invoice_id, SUM(amount_minor)::bigint AS total FROM invoice_adjustments WHERE organization_id = @organization_id AND kind = 'payment_reversal' AND status = 'approved' GROUP BY invoice_id) reversed ON reversed.invoice_id = p.invoice_id
			WHERE p.organization_id = @organization_id GROUP BY p.invoice_id
		), credits AS (
			SELECT invoice_id, SUM(amount_minor)::bigint AS credit_minor FROM invoice_adjustments WHERE organization_id = @organization_id AND kind = 'credit_note' AND status = 'approved' GROUP BY invoice_id
		), open_invoices AS (
			SELECT GREATEST(i.amount_minor - COALESCE(c.credit_minor, 0) - COALESCE(p.paid_minor, 0), 0)::bigint AS balance,
				CASE WHEN i.due_on >= CURRENT_DATE THEN 'Current' WHEN CURRENT_DATE - i.due_on <= 30 THEN '1–30 days' WHEN CURRENT_DATE - i.due_on <= 60 THEN '31–60 days' ELSE '61+ days' END AS bucket,
				CASE WHEN i.due_on >= CURRENT_DATE THEN 1 WHEN CURRENT_DATE - i.due_on <= 30 THEN 2 WHEN CURRENT_DATE - i.due_on <= 60 THEN 3 ELSE 4 END AS position
			FROM invoices i LEFT JOIN paid p ON p.invoice_id = i.id LEFT JOIN credits c ON c.invoice_id = i.id
			WHERE i.organization_id = @organization_id AND i.status IN ('open', 'partial', 'overdue')
		)
		SELECT bucket, SUM(balance)::bigint AS amount_minor
		FROM open_invoices GROUP BY bucket, position ORDER BY position`,
		sql.Named("organization_id", organizationID),
	).Scan(&buckets).Error
	if err != nil {
		return nil, fmt.Errorf("load receivables aging: %w", err)
	}
	return buckets, nil
}
