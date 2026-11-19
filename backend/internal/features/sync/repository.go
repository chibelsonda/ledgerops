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
	if len(runs) == 0 {
		return runs, nil
	}
	runIDs := make([]string, 0, len(runs))
	byID := make(map[string]*Run, len(runs))
	for index := range runs {
		runs[index].Entities = make([]EntityProgress, 0)
		runIDs = append(runIDs, runs[index].ID)
		byID[runs[index].ID] = &runs[index]
	}
	var rows []struct {
		SyncRunID string
		EntityProgress
	}
	if err := r.db.WithContext(ctx).Table("sync_run_entities").
		Select("sync_run_id::text AS sync_run_id, entity_type, status, records_processed, error_message").
		Where("sync_run_id IN ?", runIDs).Order("entity_type").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("list sync entity progress: %w", err)
	}
	for _, row := range rows {
		if run := byID[row.SyncRunID]; run != nil {
			run.Entities = append(run.Entities, row.EntityProgress)
		}
	}
	return runs, nil
}

func (r *PostgresRepository) Enqueue(ctx context.Context, organizationID string) (Run, error) {
	var run Run
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var lockAcquired int
		if err := tx.Raw(`SELECT 1 FROM pg_advisory_xact_lock(hashtextextended(?, 0))`,
			"quickbooks-sync:"+organizationID).Scan(&lockAcquired).Error; err != nil {
			return fmt.Errorf("lock QuickBooks sync enqueue: %w", err)
		}
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
		active := tx.Raw(`
			SELECT id::text AS id, provider, status, records_processed, error_message,
				to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at,
				CASE WHEN finished_at IS NULL THEN NULL ELSE to_char(finished_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') END AS finished_at
			FROM sync_runs
			WHERE organization_id = ? AND provider = 'quickbooks-sandbox' AND status IN ('queued', 'running')
			ORDER BY created_at DESC LIMIT 1`, organizationID).Scan(&run)
		if active.Error != nil {
			return fmt.Errorf("find active QuickBooks sync: %w", active.Error)
		}
		if active.RowsAffected > 0 && run.ID != "" {
			return nil
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

func (r *PostgresRepository) EnqueueDue(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 25
	}
	enqueued := 0
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var schedules []struct {
			OrganizationID  string
			IntervalMinutes int
		}
		if err := tx.Raw(`
			SELECT organization_id::text AS organization_id, interval_minutes
			FROM quickbooks_sync_schedules
			WHERE enabled AND next_run_at <= now()
			ORDER BY next_run_at FOR UPDATE SKIP LOCKED LIMIT ?`, limit).Scan(&schedules).Error; err != nil {
			return fmt.Errorf("claim due QuickBooks schedules: %w", err)
		}
		for _, schedule := range schedules {
			var active bool
			if err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM sync_runs WHERE organization_id = ?
				AND provider = 'quickbooks-sandbox' AND status IN ('queued', 'running'))`, schedule.OrganizationID).Scan(&active).Error; err != nil {
				return fmt.Errorf("check active scheduled sync: %w", err)
			}
			if !active {
				var runID string
				if err := tx.Raw(`INSERT INTO sync_runs (organization_id, provider, status)
					VALUES (?, 'quickbooks-sandbox', 'queued') RETURNING id::text`, schedule.OrganizationID).Scan(&runID).Error; err != nil {
					return fmt.Errorf("create scheduled sync run: %w", err)
				}
				if err := tx.Exec(`INSERT INTO sync_jobs (organization_id, sync_run_id, provider, idempotency_key)
					VALUES (?, ?, 'quickbooks-sandbox', ?)`, schedule.OrganizationID, runID,
					schedule.OrganizationID+":"+runID).Error; err != nil {
					return fmt.Errorf("enqueue scheduled sync job: %w", err)
				}
				enqueued++
			}
			if err := tx.Exec(`UPDATE quickbooks_sync_schedules SET last_enqueued_at = now(),
				next_run_at = now() + make_interval(mins => interval_minutes), updated_at = now()
				WHERE organization_id = ?`, schedule.OrganizationID).Error; err != nil {
				return fmt.Errorf("advance QuickBooks schedule: %w", err)
			}
		}
		return nil
	})
	return enqueued, err
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

func (r *PostgresRepository) CompleteJob(ctx context.Context, job Job, recordsProcessed int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("sync_jobs").Where("id = ?", job.ID).Updates(map[string]any{
			"status":     "succeeded",
			"locked_at":  nil,
			"updated_at": gorm.Expr("now()"),
		}).Error; err != nil {
			return fmt.Errorf("complete sync job: %w", err)
		}
		if err := tx.Table("sync_runs").Where("id = ?", job.SyncRunID).Updates(map[string]any{
			"status":            "succeeded",
			"records_processed": recordsProcessed,
			"error_message":     nil,
			"finished_at":       gorm.Expr("now()"),
		}).Error; err != nil {
			return fmt.Errorf("complete sync run: %w", err)
		}
		return nil
	})
}

func (r *PostgresRepository) FailJob(ctx context.Context, job Job, cause error, retryable bool, retryAfter time.Duration) error {
	backoff := retryAfter
	if backoff <= 0 {
		backoff = time.Duration(job.Attempts*job.Attempts) * time.Minute
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			UPDATE sync_jobs SET status = CASE WHEN ? AND attempts < max_attempts THEN 'queued' ELSE 'failed' END,
				last_error = ?, locked_at = NULL, next_attempt_at = now() + ?::interval, updated_at = now()
			WHERE id = ?`, retryable, cause.Error(), backoff.String(), job.ID).Error; err != nil {
			return fmt.Errorf("fail sync job: %w", err)
		}
		if err := tx.Exec(`
			UPDATE sync_runs SET status = CASE
					WHEN ? AND EXISTS (SELECT 1 FROM sync_jobs WHERE id = ? AND status = 'queued') THEN 'queued'
					ELSE 'failed' END,
				error_message = ?, finished_at = CASE
					WHEN ? AND EXISTS (SELECT 1 FROM sync_jobs WHERE id = ? AND status = 'queued') THEN NULL
					ELSE now() END
			WHERE id = ?`, retryable, job.ID, cause.Error(), retryable, job.ID, job.SyncRunID,
		).Error; err != nil {
			return fmt.Errorf("fail sync run: %w", err)
		}
		return nil
	})
}
