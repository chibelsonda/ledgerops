package reconciliation

import (
	"context"
	"encoding/json"
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

func (r *PostgresRepository) List(ctx context.Context, organizationID string) ([]Item, error) {
	items := make([]Item, 0)
	err := r.db.WithContext(ctx).Raw(`
		SELECT id::text AS id, kind, severity, title, description, amount_minor, currency_code, status,
			to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at
		FROM reconciliation_items
		WHERE organization_id = ?
		ORDER BY CASE status WHEN 'open' THEN 0 ELSE 1 END, created_at DESC`, organizationID).Scan(&items).Error
	if err != nil {
		return nil, fmt.Errorf("list reconciliation items: %w", err)
	}
	return items, nil
}

func (r *PostgresRepository) Resolve(ctx context.Context, organizationID, itemID, actor, note string) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Table("reconciliation_items").
			Where("id = ? AND organization_id = ? AND status = ?", itemID, organizationID, "open").
			Updates(map[string]any{
				"status":        "resolved",
				"resolved_note": note,
				"resolved_at":   gorm.Expr("now()"),
			})
		if result.Error != nil {
			return fmt.Errorf("resolve reconciliation item: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return apperrors.ErrNotFound
		}

		metadata, err := json.Marshal(map[string]string{"note": note})
		if err != nil {
			return fmt.Errorf("encode reconciliation audit metadata: %w", err)
		}
		if err := tx.Exec(`
			INSERT INTO audit_events (organization_id, actor, action, entity_type, entity_id, metadata)
			VALUES (?, ?, 'reconciliation.resolved', 'reconciliation_item', ?, ?::jsonb)`,
			organizationID, actor, itemID, string(metadata),
		).Error; err != nil {
			return fmt.Errorf("write reconciliation audit event: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("commit reconciliation resolution: %w", err)
	}
	return nil
}
