package close

import (
	"context"
	"errors"
	"fmt"

	"github.com/portfolio/ledgerops/backend/internal/platform/apperrors"
	"github.com/portfolio/ledgerops/backend/internal/platform/audit"
	"gorm.io/gorm"
)

type PostgresRepository struct{ db *gorm.DB }

func NewPostgresRepository(db *gorm.DB) *PostgresRepository { return &PostgresRepository{db: db} }

const periodSummaryProjection = `
	SELECT cp.id::text AS id,to_char(cp.month_start,'YYYY-MM-DD') AS month_start,to_char(cp.due_on,'YYYY-MM-DD') AS due_on,cp.status,
		cp.created_by::text AS created_by_id,creator.email AS created_by,
		COALESCE(cp.submitted_by::text,'') AS submitted_by_id,COALESCE(submitter.email,'') AS submitted_by,
		COALESCE(reviewer.email,'') AS reviewed_by,COALESCE(cp.review_note,'') AS review_note,
		COUNT(ct.id)::int AS total_tasks,COUNT(ct.id) FILTER (WHERE ct.status='completed')::int AS completed_tasks,
		CASE WHEN COUNT(ct.id)=0 THEN 0 ELSE ROUND(COUNT(ct.id) FILTER (WHERE ct.status='completed') * 100.0 / COUNT(ct.id))::int END AS progress,
		(SELECT COUNT(*)::int FROM reconciliation_items ri WHERE ri.organization_id=cp.organization_id AND ri.status='open') AS open_exceptions,
		COALESCE(to_char(cp.submitted_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),'') AS submitted_at,
		COALESCE(to_char(cp.reviewed_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),'') AS reviewed_at,
		COALESCE(to_char(cp.closed_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),'') AS closed_at,
		to_char(cp.created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at
	FROM close_periods cp JOIN users creator ON creator.id=cp.created_by
	LEFT JOIN users submitter ON submitter.id=cp.submitted_by LEFT JOIN users reviewer ON reviewer.id=cp.reviewed_by
	LEFT JOIN close_tasks ct ON ct.close_period_id=cp.id`

const periodSummaryGroup = ` GROUP BY cp.id,creator.email,submitter.email,reviewer.email`

func (r *PostgresRepository) List(ctx context.Context, organizationID string) ([]PeriodSummary, error) {
	items := make([]PeriodSummary, 0)
	if err := r.db.WithContext(ctx).Raw(periodSummaryProjection+` WHERE cp.organization_id=?`+periodSummaryGroup+` ORDER BY cp.month_start DESC`, organizationID).Scan(&items).Error; err != nil {
		return nil, fmt.Errorf("list close periods: %w", err)
	}
	return items, nil
}

func (r *PostgresRepository) Get(ctx context.Context, organizationID, periodID string) (Detail, error) {
	var detail Detail
	if err := r.db.WithContext(ctx).Raw(periodSummaryProjection+` WHERE cp.organization_id=? AND cp.id=?`+periodSummaryGroup, organizationID, periodID).Scan(&detail.Period).Error; err != nil {
		return detail, fmt.Errorf("get close period: %w", err)
	}
	if detail.Period.ID == "" {
		return detail, apperrors.ErrNotFound
	}
	detail.Tasks = make([]Task, 0)
	if err := r.db.WithContext(ctx).Raw(`
		SELECT ct.id::text AS id,ct.title,ct.category,ct.position,ct.status,COALESCE(ct.evidence_note,'') AS evidence_note,
			COALESCE(u.email,'') AS completed_by,COALESCE(to_char(ct.completed_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),'') AS completed_at
		FROM close_tasks ct LEFT JOIN users u ON u.id=ct.completed_by
		WHERE ct.organization_id=? AND ct.close_period_id=? ORDER BY ct.position`, organizationID, periodID).Scan(&detail.Tasks).Error; err != nil {
		return detail, fmt.Errorf("list close tasks: %w", err)
	}
	events, err := r.listAudit(ctx, organizationID, periodID)
	if err != nil {
		return detail, err
	}
	detail.AuditEvents = events
	return detail, nil
}

var defaultTasks = []struct{ title, category string }{
	{"Reconcile operating bank accounts", "Cash"},
	{"Review receivables and credit notes", "Revenue"},
	{"Record accruals and vendor obligations", "Expenses"},
	{"Verify payroll and operating expenses", "Expenses"},
	{"Confirm project billing completeness", "Projects"},
	{"Prepare management variance review", "Reporting"},
}

func (r *PostgresRepository) Create(ctx context.Context, organizationID string, actor Actor, input CreateInput) (Detail, error) {
	var periodID string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row struct{ ID string }
		result := tx.Raw(`INSERT INTO close_periods (organization_id,month_start,due_on,created_by) VALUES (?,?::date,?::date,?) RETURNING id::text AS id`, organizationID, input.MonthStart, input.DueOn, actor.UserID).Scan(&row)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
				return apperrors.ErrConflict
			}
			return fmt.Errorf("insert close period: %w", result.Error)
		}
		periodID = row.ID
		for index, task := range defaultTasks {
			if err := tx.Exec(`INSERT INTO close_tasks (organization_id,close_period_id,title,category,position) VALUES (?,?,?,?,?)`, organizationID, periodID, task.title, task.category, index+1).Error; err != nil {
				return fmt.Errorf("insert close task: %w", err)
			}
		}
		return writeAudit(tx, organizationID, actor.Email, "close_period.created", periodID, map[string]any{"monthStart": input.MonthStart, "dueOn": input.DueOn})
	})
	if err != nil {
		return Detail{}, fmt.Errorf("create close period transaction: %w", err)
	}
	return r.Get(ctx, organizationID, periodID)
}

func (r *PostgresRepository) UpdateTask(ctx context.Context, organizationID, taskID string, actor Actor, input TaskInput) (Detail, error) {
	var periodID string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task struct{ ID, PeriodID, PeriodStatus, OldStatus string }
		if err := tx.Raw(`SELECT ct.id::text AS id,ct.close_period_id::text AS period_id,cp.status AS period_status,ct.status AS old_status FROM close_tasks ct JOIN close_periods cp ON cp.id=ct.close_period_id WHERE ct.id=? AND ct.organization_id=? FOR UPDATE OF ct,cp`, taskID, organizationID).Scan(&task).Error; err != nil {
			return fmt.Errorf("lock close task: %w", err)
		}
		if task.ID == "" {
			return apperrors.ErrNotFound
		}
		if task.PeriodStatus != "draft" {
			return apperrors.ErrInvalidState
		}
		periodID = task.PeriodID
		if input.Status == "completed" {
			if err := tx.Exec(`UPDATE close_tasks SET status='completed',evidence_note=?,completed_by=?,completed_at=now(),updated_at=now() WHERE id=?`, input.EvidenceNote, actor.UserID, taskID).Error; err != nil {
				return fmt.Errorf("complete close task: %w", err)
			}
		} else if err := tx.Exec(`UPDATE close_tasks SET status='pending',evidence_note=NULL,completed_by=NULL,completed_at=NULL,updated_at=now() WHERE id=?`, taskID).Error; err != nil {
			return fmt.Errorf("reopen close task: %w", err)
		}
		return writeAudit(tx, organizationID, actor.Email, "close_task."+input.Status, periodID, map[string]any{"taskId": taskID, "evidence": input.EvidenceNote})
	})
	if err != nil {
		return Detail{}, fmt.Errorf("update close task transaction: %w", err)
	}
	return r.Get(ctx, organizationID, periodID)
}

func (r *PostgresRepository) Submit(ctx context.Context, organizationID, periodID string, actor Actor) (Detail, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var period struct {
			ID, Status string
			Pending    int
		}
		if err := tx.Raw(`SELECT cp.id::text AS id,cp.status,(SELECT COUNT(*)::int FROM close_tasks WHERE close_period_id=cp.id AND status<>'completed') AS pending FROM close_periods cp WHERE cp.id=? AND cp.organization_id=? FOR UPDATE`, periodID, organizationID).Scan(&period).Error; err != nil {
			return fmt.Errorf("lock close period: %w", err)
		}
		if period.ID == "" {
			return apperrors.ErrNotFound
		}
		if period.Status != "draft" || period.Pending != 0 {
			return apperrors.ErrInvalidState
		}
		if err := tx.Exec(`UPDATE close_periods SET status='in_review',submitted_by=?,submitted_at=now(),updated_at=now() WHERE id=?`, actor.UserID, periodID).Error; err != nil {
			return fmt.Errorf("submit close period: %w", err)
		}
		return writeAudit(tx, organizationID, actor.Email, "close_period.submitted", periodID, map[string]any{})
	})
	if err != nil {
		return Detail{}, fmt.Errorf("submit close period transaction: %w", err)
	}
	return r.Get(ctx, organizationID, periodID)
}

func (r *PostgresRepository) Decide(ctx context.Context, organizationID, periodID string, actor Actor, input DecisionInput) (Detail, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var period struct{ ID, Status, SubmittedByID string }
		if err := tx.Raw(`SELECT id::text AS id,status,COALESCE(submitted_by::text,'') AS submitted_by_id FROM close_periods WHERE id=? AND organization_id=? FOR UPDATE`, periodID, organizationID).Scan(&period).Error; err != nil {
			return fmt.Errorf("lock close period review: %w", err)
		}
		if period.ID == "" {
			return apperrors.ErrNotFound
		}
		if period.Status != "in_review" || period.SubmittedByID == actor.UserID {
			return apperrors.ErrInvalidState
		}
		status := "draft"
		closedAt := "NULL"
		action := "close_period.rejected"
		if input.Decision == "approved" {
			status, closedAt, action = "closed", "now()", "close_period.closed"
		}
		if err := tx.Exec(`UPDATE close_periods SET status=?,reviewed_by=?,review_note=?,reviewed_at=now(),closed_at=`+closedAt+`,updated_at=now() WHERE id=?`, status, actor.UserID, input.Note, periodID).Error; err != nil {
			return fmt.Errorf("decide close period: %w", err)
		}
		return writeAudit(tx, organizationID, actor.Email, action, periodID, map[string]any{"note": input.Note})
	})
	if err != nil {
		return Detail{}, fmt.Errorf("decide close period transaction: %w", err)
	}
	return r.Get(ctx, organizationID, periodID)
}

func (r *PostgresRepository) listAudit(ctx context.Context, organizationID, periodID string) ([]AuditEvent, error) {
	type row struct{ ID, Actor, Action, Metadata, CreatedAt string }
	rows := make([]row, 0)
	if err := r.db.WithContext(ctx).Raw(`SELECT id::text AS id,actor,action,metadata::text AS metadata,to_char(created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at FROM audit_events WHERE organization_id=? AND entity_type='close_period' AND entity_id=? ORDER BY created_at DESC`, organizationID, periodID).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("list close audit events: %w", err)
	}
	events := make([]AuditEvent, 0, len(rows))
	for _, item := range rows {
		metadata, err := audit.DecodeMetadata(item.Metadata)
		if err != nil {
			return nil, fmt.Errorf("decode close audit metadata: %w", err)
		}
		events = append(events, AuditEvent{ID: item.ID, Actor: item.Actor, Action: item.Action, Metadata: metadata, CreatedAt: item.CreatedAt})
	}
	return events, nil
}

func writeAudit(tx *gorm.DB, organizationID, actor, action, periodID string, metadata map[string]any) error {
	return audit.Write(tx, organizationID, actor, action, "close_period", periodID, metadata)
}
