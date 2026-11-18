package sync

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

var ErrProviderNotConnected = errors.New("sync provider is not connected")

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) ListRuns(ctx context.Context, organizationID string) ([]Run, error) {
	runs := make([]Run, 0)
	err := r.db.WithContext(ctx).Raw(`
		SELECT id::text AS id, provider, status, records_processed, error_message,
			to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at,
			CASE WHEN finished_at IS NULL THEN NULL ELSE to_char(finished_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') END AS finished_at
		FROM sync_runs WHERE organization_id = ? ORDER BY created_at DESC LIMIT 10`, organizationID).Scan(&runs).Error
	if err != nil {
		return nil, fmt.Errorf("list sync runs: %w", err)
	}
	return runs, nil
}

func (r *PostgresRepository) Enqueue(ctx context.Context, organizationID string) (Run, error) {
	var run Run
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var connected bool
		if err := tx.Raw(`
			SELECT EXISTS(
				SELECT 1 FROM quickbooks_connections WHERE organization_id = ?
			)`, organizationID).Scan(&connected).Error; err != nil {
			return fmt.Errorf("check QuickBooks connection: %w", err)
		}
		if !connected {
			return ErrProviderNotConnected
		}
		if err := tx.Raw(`
			INSERT INTO sync_runs (organization_id, provider, status) VALUES (?, 'quickbooks-sandbox', 'queued')
			RETURNING id::text AS id, provider, status, records_processed,
				to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at`,
			organizationID,
		).Scan(&run).Error; err != nil {
			return fmt.Errorf("create sync run: %w", err)
		}

		idempotencyKey := fmt.Sprintf("%s:%s", organizationID, run.ID)
		if err := tx.Exec(`
			INSERT INTO sync_jobs (organization_id, sync_run_id, provider, idempotency_key)
			VALUES (?, ?, 'quickbooks-sandbox', ?)`, organizationID, run.ID, idempotencyKey,
		).Error; err != nil {
			return fmt.Errorf("enqueue sync job: %w", err)
		}
		return nil
	})
	if err != nil {
		return Run{}, fmt.Errorf("commit sync: %w", err)
	}
	return run, nil
}

func (r *PostgresRepository) ClaimJob(ctx context.Context) (*Job, error) {
	var job Job
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Raw(`
			WITH candidate AS (
				SELECT id FROM sync_jobs WHERE status = 'queued' AND next_attempt_at <= now()
				ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1
			)
			UPDATE sync_jobs j SET status = 'running', locked_at = now(), attempts = attempts + 1, updated_at = now()
			FROM candidate c WHERE j.id = c.id
			RETURNING j.id::text AS id, j.organization_id::text AS organization_id,
				j.sync_run_id::text AS sync_run_id, j.provider, j.attempts`,
		).Scan(&job)
		if result.Error != nil {
			return fmt.Errorf("claim sync job: %w", result.Error)
		}
		if result.RowsAffected == 0 || job.ID == "" {
			return nil
		}
		if err := tx.Table("sync_runs").Where("id = ?", job.SyncRunID).Updates(map[string]any{
			"status":     "running",
			"started_at": gorm.Expr("COALESCE(started_at, now())"),
		}).Error; err != nil {
			return fmt.Errorf("mark sync run running: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if job.ID == "" {
		return nil, nil
	}
	return &job, nil
}

func (r *PostgresRepository) CompleteJob(ctx context.Context, job Job) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var records int
		if err := tx.Raw(`
			SELECT (SELECT COUNT(*) FROM clients WHERE organization_id = ?)
				 + (SELECT COUNT(*) FROM invoices WHERE organization_id = ?)
				 + (SELECT COUNT(*) FROM payments WHERE organization_id = ?)
				 + (SELECT COUNT(*) FROM expenses WHERE organization_id = ?)`,
			job.OrganizationID, job.OrganizationID, job.OrganizationID, job.OrganizationID,
		).Scan(&records).Error; err != nil {
			return fmt.Errorf("count synchronized records: %w", err)
		}
		if err := tx.Table("sync_jobs").Where("id = ?", job.ID).Updates(map[string]any{
			"status":     "succeeded",
			"locked_at":  nil,
			"updated_at": gorm.Expr("now()"),
		}).Error; err != nil {
			return fmt.Errorf("complete sync job: %w", err)
		}
		if err := tx.Table("sync_runs").Where("id = ?", job.SyncRunID).Updates(map[string]any{
			"status":            "succeeded",
			"records_processed": records,
			"finished_at":       gorm.Expr("now()"),
		}).Error; err != nil {
			return fmt.Errorf("complete sync run: %w", err)
		}
		return nil
	})
}

func (r *PostgresRepository) FailJob(ctx context.Context, job Job, cause error) error {
	backoff := time.Duration(job.Attempts*job.Attempts) * time.Minute
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			UPDATE sync_jobs SET status = CASE WHEN attempts >= max_attempts THEN 'failed' ELSE 'queued' END,
				last_error = ?, locked_at = NULL, next_attempt_at = now() + ?::interval, updated_at = now()
			WHERE id = ?`, cause.Error(), backoff.String(), job.ID).Error; err != nil {
			return fmt.Errorf("fail sync job: %w", err)
		}
		if err := tx.Exec(`
			UPDATE sync_runs SET status = CASE WHEN ? >= 5 THEN 'failed' ELSE 'queued' END,
				error_message = ?, finished_at = CASE WHEN ? >= 5 THEN now() ELSE NULL END WHERE id = ?`,
			job.Attempts, cause.Error(), job.Attempts, job.SyncRunID,
		).Error; err != nil {
			return fmt.Errorf("fail sync run: %w", err)
		}
		return nil
	})
}
