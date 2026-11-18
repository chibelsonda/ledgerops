package projects

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/portfolio/ledgerops/backend/internal/platform/apperrors"
	"gorm.io/gorm"
)

type PostgresRepository struct{ db *gorm.DB }

func NewPostgresRepository(db *gorm.DB) *PostgresRepository { return &PostgresRepository{db: db} }

const projectSummaryProjection = `
	SELECT p.id::text AS id,p.project_code,p.name,p.client_id::text AS client_id,c.name AS client_name,
		p.status,p.currency_code,to_char(p.start_on,'YYYY-MM-DD') AS start_on,
		COALESCE(to_char(p.end_on,'YYYY-MM-DD'),'') AS end_on,
		pc.id::text AS contract_id,pc.contract_number,pc.status AS contract_status,pc.original_value_minor,
		COALESCE((SELECT SUM(co.amount_minor) FROM contract_change_orders co WHERE co.contract_id=pc.id AND co.status='approved'),0)::bigint AS approved_changes_minor,
		(pc.original_value_minor + COALESCE((SELECT SUM(co.amount_minor) FROM contract_change_orders co WHERE co.contract_id=pc.id AND co.status='approved'),0))::bigint AS revised_value_minor,
		COALESCE((SELECT SUM(m.amount_minor) FROM billing_milestones m WHERE m.contract_id=pc.id),0)::bigint AS allocated_minor,
		COALESCE((SELECT SUM(m.amount_minor) FROM billing_milestones m WHERE m.contract_id=pc.id AND m.status='invoiced'),0)::bigint AS billed_minor,
		(pc.original_value_minor
			+ COALESCE((SELECT SUM(co.amount_minor) FROM contract_change_orders co WHERE co.contract_id=pc.id AND co.status='approved'),0)
			- COALESCE((SELECT SUM(m.amount_minor) FROM billing_milestones m WHERE m.contract_id=pc.id),0))::bigint AS unallocated_minor,
		to_char(p.created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at
	FROM projects p JOIN clients c ON c.id=p.client_id JOIN project_contracts pc ON pc.project_id=p.id`

func (r *PostgresRepository) List(ctx context.Context, organizationID string) ([]ProjectSummary, error) {
	items := make([]ProjectSummary, 0)
	if err := r.db.WithContext(ctx).Raw(projectSummaryProjection+`
		WHERE p.organization_id=? ORDER BY p.created_at DESC,p.id DESC`, organizationID).Scan(&items).Error; err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return items, nil
}

func (r *PostgresRepository) ListClients(ctx context.Context, organizationID string) ([]ClientOption, error) {
	items := make([]ClientOption, 0)
	if err := r.db.WithContext(ctx).Raw(`SELECT id::text AS id,name FROM clients WHERE organization_id=? ORDER BY name`, organizationID).Scan(&items).Error; err != nil {
		return nil, fmt.Errorf("list project clients: %w", err)
	}
	return items, nil
}

func (r *PostgresRepository) Get(ctx context.Context, organizationID, projectID string) (Detail, error) {
	var detail Detail
	if err := r.db.WithContext(ctx).Raw(projectSummaryProjection+`
		WHERE p.organization_id=? AND p.id=?`, organizationID, projectID).Scan(&detail.Project).Error; err != nil {
		return detail, fmt.Errorf("get project summary: %w", err)
	}
	if detail.Project.ID == "" {
		return detail, apperrors.ErrNotFound
	}
	if err := r.db.WithContext(ctx).Raw(`
		SELECT pc.id::text AS id,pc.contract_number,pc.title,pc.original_value_minor,pc.status,
			pc.requested_by::text AS requested_by_id,requester.email AS requested_by,
			COALESCE(reviewer.email,'') AS reviewed_by,COALESCE(pc.review_note,'') AS review_note,
			to_char(pc.requested_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') AS requested_at,
			COALESCE(to_char(pc.reviewed_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),'') AS reviewed_at
		FROM project_contracts pc JOIN users requester ON requester.id=pc.requested_by
		LEFT JOIN users reviewer ON reviewer.id=pc.reviewed_by
		WHERE pc.organization_id=? AND pc.project_id=?`, organizationID, projectID).Scan(&detail.Contract).Error; err != nil {
		return detail, fmt.Errorf("get project contract: %w", err)
	}
	detail.ChangeOrders = make([]ChangeOrder, 0)
	if err := r.db.WithContext(ctx).Raw(`
		SELECT co.id::text AS id,co.change_order_number,co.title,co.description,co.amount_minor,co.status,
			co.requested_by::text AS requested_by_id,requester.email AS requested_by,
			COALESCE(reviewer.email,'') AS reviewed_by,COALESCE(co.review_note,'') AS review_note,
			to_char(co.requested_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') AS requested_at,
			COALESCE(to_char(co.reviewed_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),'') AS reviewed_at
		FROM contract_change_orders co JOIN project_contracts pc ON pc.id=co.contract_id
		JOIN users requester ON requester.id=co.requested_by LEFT JOIN users reviewer ON reviewer.id=co.reviewed_by
		WHERE co.organization_id=? AND pc.project_id=? ORDER BY co.requested_at DESC`, organizationID, projectID).Scan(&detail.ChangeOrders).Error; err != nil {
		return detail, fmt.Errorf("list project change orders: %w", err)
	}
	detail.Milestones = make([]BillingMilestone, 0)
	if err := r.db.WithContext(ctx).Raw(`
		SELECT m.id::text AS id,m.name,m.amount_minor,to_char(m.due_on,'YYYY-MM-DD') AS due_on,m.status,
			COALESCE(m.invoice_id::text,'') AS invoice_id,COALESCE(i.invoice_number,'') AS invoice_number,
			u.email AS created_by,to_char(m.created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at,
			COALESCE(to_char(m.invoiced_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),'') AS invoiced_at
		FROM billing_milestones m JOIN project_contracts pc ON pc.id=m.contract_id JOIN users u ON u.id=m.created_by
		LEFT JOIN invoices i ON i.id=m.invoice_id
		WHERE m.organization_id=? AND pc.project_id=? ORDER BY m.due_on,m.created_at`, organizationID, projectID).Scan(&detail.Milestones).Error; err != nil {
		return detail, fmt.Errorf("list billing milestones: %w", err)
	}
	events, err := r.listAudit(ctx, organizationID, projectID)
	if err != nil {
		return detail, err
	}
	detail.AuditEvents = events
	return detail, nil
}

func (r *PostgresRepository) Create(ctx context.Context, organizationID string, actor Actor, input CreateInput) (Detail, error) {
	var projectID string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var clientExists bool
		if err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM clients WHERE id=? AND organization_id=?)`, input.ClientID, organizationID).Scan(&clientExists).Error; err != nil {
			return fmt.Errorf("check project client: %w", err)
		}
		if !clientExists {
			return apperrors.ErrInvalidReference
		}
		var projectRow struct{ ID string }
		result := tx.Raw(`
			INSERT INTO projects (organization_id,client_id,project_code,name,currency_code,start_on,end_on,created_by)
			VALUES (?,?,?,?,?,?::date,NULLIF(?,'')::date,?) RETURNING id::text AS id`,
			organizationID, input.ClientID, input.ProjectCode, input.Name, input.CurrencyCode, input.StartOn, input.EndOn, actor.UserID).Scan(&projectRow)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
				return apperrors.ErrConflict
			}
			return fmt.Errorf("insert project: %w", result.Error)
		}
		projectID = projectRow.ID
		var contractRow struct{ ID string }
		result = tx.Raw(`
			INSERT INTO project_contracts (organization_id,project_id,contract_number,title,original_value_minor,requested_by)
			VALUES (?,?,?,?,?,?) RETURNING id::text AS id`, organizationID, projectID, input.ContractNumber, input.ContractTitle, input.OriginalValueMinor, actor.UserID).Scan(&contractRow)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
				return apperrors.ErrConflict
			}
			return fmt.Errorf("insert project contract: %w", result.Error)
		}
		return writeProjectAudit(tx, organizationID, actor.Email, "project.created", projectID, map[string]any{
			"projectCode": input.ProjectCode, "contractId": contractRow.ID, "contractNumber": input.ContractNumber,
			"originalValueMinor": input.OriginalValueMinor,
		})
	})
	if err != nil {
		return Detail{}, fmt.Errorf("create project transaction: %w", err)
	}
	return r.Get(ctx, organizationID, projectID)
}

type lockedContract struct {
	ID                   string
	ProjectID            string
	Status               string
	RequestedByID        string
	OriginalValueMinor   int64
	ApprovedChangesMinor int64
	AllocatedMinor       int64
}

func lockContract(tx *gorm.DB, organizationID, contractID string) (lockedContract, error) {
	var contract lockedContract
	if err := tx.Raw(`
		SELECT pc.id::text AS id,pc.project_id::text AS project_id,pc.status,
			pc.requested_by::text AS requested_by_id,pc.original_value_minor,
			COALESCE((SELECT SUM(amount_minor) FROM contract_change_orders WHERE contract_id=pc.id AND status='approved'),0)::bigint AS approved_changes_minor,
			COALESCE((SELECT SUM(amount_minor) FROM billing_milestones WHERE contract_id=pc.id),0)::bigint AS allocated_minor
		FROM project_contracts pc WHERE pc.id=? AND pc.organization_id=? FOR UPDATE`, contractID, organizationID).Scan(&contract).Error; err != nil {
		return contract, fmt.Errorf("lock project contract: %w", err)
	}
	if contract.ID == "" {
		return contract, apperrors.ErrNotFound
	}
	return contract, nil
}

func (r *PostgresRepository) DecideContract(ctx context.Context, organizationID, contractID string, actor Actor, input DecisionInput) (Detail, error) {
	var projectID string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		contract, err := lockContract(tx, organizationID, contractID)
		if err != nil {
			return err
		}
		if contract.Status != "pending" || contract.RequestedByID == actor.UserID {
			return apperrors.ErrInvalidState
		}
		projectID = contract.ProjectID
		if err := tx.Exec(`UPDATE project_contracts SET status=?,reviewed_by=?,review_note=?,reviewed_at=now() WHERE id=?`, input.Decision, actor.UserID, input.Note, contractID).Error; err != nil {
			return fmt.Errorf("decide project contract: %w", err)
		}
		return writeProjectAudit(tx, organizationID, actor.Email, "contract."+input.Decision, projectID, map[string]any{
			"contractId": contractID, "note": input.Note,
		})
	})
	if err != nil {
		return Detail{}, fmt.Errorf("decide contract transaction: %w", err)
	}
	return r.Get(ctx, organizationID, projectID)
}

func (r *PostgresRepository) CreateChangeOrder(ctx context.Context, organizationID, contractID string, actor Actor, input ChangeOrderInput) (Detail, error) {
	var projectID string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		contract, err := lockContract(tx, organizationID, contractID)
		if err != nil {
			return err
		}
		if contract.Status != "approved" || contract.OriginalValueMinor+contract.ApprovedChangesMinor+input.AmountMinor <= 0 || contract.OriginalValueMinor+contract.ApprovedChangesMinor+input.AmountMinor < contract.AllocatedMinor {
			return apperrors.ErrInvalidState
		}
		projectID = contract.ProjectID
		var row struct{ ID string }
		result := tx.Raw(`
			INSERT INTO contract_change_orders (organization_id,contract_id,change_order_number,title,description,amount_minor,requested_by)
			VALUES (?,?,?,?,?,?,?) RETURNING id::text AS id`, organizationID, contractID, input.ChangeOrderNumber, input.Title, input.Description, input.AmountMinor, actor.UserID).Scan(&row)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
				return apperrors.ErrConflict
			}
			return fmt.Errorf("insert change order: %w", result.Error)
		}
		return writeProjectAudit(tx, organizationID, actor.Email, "change_order.requested", projectID, map[string]any{
			"changeOrderId": row.ID, "number": input.ChangeOrderNumber, "amountMinor": input.AmountMinor,
		})
	})
	if err != nil {
		return Detail{}, fmt.Errorf("create change order transaction: %w", err)
	}
	return r.Get(ctx, organizationID, projectID)
}

func (r *PostgresRepository) DecideChangeOrder(ctx context.Context, organizationID, changeOrderID string, actor Actor, input DecisionInput) (Detail, error) {
	var projectID string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var lookup struct{ ContractID string }
		if err := tx.Raw(`SELECT contract_id::text AS contract_id FROM contract_change_orders WHERE id=? AND organization_id=?`, changeOrderID, organizationID).Scan(&lookup).Error; err != nil {
			return fmt.Errorf("find change-order contract: %w", err)
		}
		if lookup.ContractID == "" {
			return apperrors.ErrNotFound
		}
		contract, err := lockContract(tx, organizationID, lookup.ContractID)
		if err != nil {
			return err
		}
		var order struct {
			ID            string
			Status        string
			RequestedByID string
			AmountMinor   int64
		}
		if err := tx.Raw(`SELECT id::text AS id,status,requested_by::text AS requested_by_id,amount_minor FROM contract_change_orders WHERE id=? AND organization_id=? FOR UPDATE`, changeOrderID, organizationID).Scan(&order).Error; err != nil {
			return fmt.Errorf("lock change order: %w", err)
		}
		if order.ID == "" {
			return apperrors.ErrNotFound
		}
		if contract.Status != "approved" || order.Status != "pending" || order.RequestedByID == actor.UserID {
			return apperrors.ErrInvalidState
		}
		if input.Decision == "approved" {
			revised := contract.OriginalValueMinor + contract.ApprovedChangesMinor + order.AmountMinor
			if revised <= 0 || revised < contract.AllocatedMinor {
				return apperrors.ErrInvalidState
			}
		}
		projectID = contract.ProjectID
		if err := tx.Exec(`UPDATE contract_change_orders SET status=?,reviewed_by=?,review_note=?,reviewed_at=now() WHERE id=?`, input.Decision, actor.UserID, input.Note, changeOrderID).Error; err != nil {
			return fmt.Errorf("decide change order: %w", err)
		}
		return writeProjectAudit(tx, organizationID, actor.Email, "change_order."+input.Decision, projectID, map[string]any{
			"changeOrderId": changeOrderID, "amountMinor": order.AmountMinor, "note": input.Note,
		})
	})
	if err != nil {
		return Detail{}, fmt.Errorf("decide change order transaction: %w", err)
	}
	return r.Get(ctx, organizationID, projectID)
}

func (r *PostgresRepository) CreateMilestone(ctx context.Context, organizationID, contractID string, actor Actor, input MilestoneInput) (Detail, error) {
	var projectID string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		contract, err := lockContract(tx, organizationID, contractID)
		if err != nil {
			return err
		}
		revised := contract.OriginalValueMinor + contract.ApprovedChangesMinor
		if contract.Status != "approved" || contract.AllocatedMinor+input.AmountMinor > revised {
			return apperrors.ErrInvalidState
		}
		projectID = contract.ProjectID
		var row struct{ ID string }
		if err := tx.Raw(`
			INSERT INTO billing_milestones (organization_id,contract_id,name,amount_minor,due_on,created_by)
			VALUES (?,?,?,?,?::date,?) RETURNING id::text AS id`, organizationID, contractID, input.Name, input.AmountMinor, input.DueOn, actor.UserID).Scan(&row).Error; err != nil {
			return fmt.Errorf("insert billing milestone: %w", err)
		}
		return writeProjectAudit(tx, organizationID, actor.Email, "milestone.created", projectID, map[string]any{
			"milestoneId": row.ID, "name": input.Name, "amountMinor": input.AmountMinor, "dueOn": input.DueOn,
		})
	})
	if err != nil {
		return Detail{}, fmt.Errorf("create milestone transaction: %w", err)
	}
	return r.Get(ctx, organizationID, projectID)
}

func (r *PostgresRepository) InvoiceMilestone(ctx context.Context, organizationID, milestoneID string, actor Actor, input InvoiceInput) (Detail, error) {
	var projectID string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var milestone struct {
			ID             string
			ContractID     string
			ProjectID      string
			ClientID       string
			ContractStatus string
			Status         string
			CurrencyCode   string
			AmountMinor    int64
		}
		if err := tx.Raw(`
			SELECT m.id::text AS id,m.contract_id::text AS contract_id,pc.project_id::text AS project_id,
				p.client_id::text AS client_id,pc.status AS contract_status,m.status,p.currency_code,m.amount_minor
			FROM billing_milestones m JOIN project_contracts pc ON pc.id=m.contract_id JOIN projects p ON p.id=pc.project_id
			WHERE m.id=? AND m.organization_id=? FOR UPDATE OF m,pc,p`, milestoneID, organizationID).Scan(&milestone).Error; err != nil {
			return fmt.Errorf("lock billing milestone: %w", err)
		}
		if milestone.ID == "" {
			return apperrors.ErrNotFound
		}
		if milestone.Status != "planned" || milestone.ContractStatus != "approved" {
			return apperrors.ErrInvalidState
		}
		projectID = milestone.ProjectID
		var invoiceRow struct{ ID string }
		result := tx.Raw(`
			INSERT INTO invoices (organization_id,client_id,external_id,invoice_number,status,currency_code,amount_minor,issued_on,due_on,source)
			VALUES (?,?,?, ?,CASE WHEN ?::date < CURRENT_DATE THEN 'overdue' ELSE 'open' END,?,?,?::date,?::date,'project_billing')
			RETURNING id::text AS id`, organizationID, milestone.ClientID, milestoneID, input.InvoiceNumber, input.DueOn, milestone.CurrencyCode, milestone.AmountMinor, input.IssuedOn, input.DueOn).Scan(&invoiceRow)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
				return apperrors.ErrConflict
			}
			return fmt.Errorf("insert milestone invoice: %w", result.Error)
		}
		if err := tx.Exec(`UPDATE billing_milestones SET status='invoiced',invoice_id=?,invoiced_at=now() WHERE id=?`, invoiceRow.ID, milestoneID).Error; err != nil {
			return fmt.Errorf("finalize billing milestone: %w", err)
		}
		if err := writeProjectAudit(tx, organizationID, actor.Email, "milestone.invoiced", projectID, map[string]any{
			"milestoneId": milestoneID, "invoiceId": invoiceRow.ID, "invoiceNumber": input.InvoiceNumber, "amountMinor": milestone.AmountMinor,
		}); err != nil {
			return err
		}
		return writeInvoiceAudit(tx, organizationID, actor.Email, invoiceRow.ID, map[string]any{
			"projectId": projectID, "milestoneId": milestoneID, "invoiceNumber": input.InvoiceNumber, "amountMinor": milestone.AmountMinor,
		})
	})
	if err != nil {
		return Detail{}, fmt.Errorf("invoice milestone transaction: %w", err)
	}
	return r.Get(ctx, organizationID, projectID)
}

func (r *PostgresRepository) listAudit(ctx context.Context, organizationID, projectID string) ([]AuditEvent, error) {
	type row struct {
		ID        string
		Actor     string
		Action    string
		Metadata  string
		CreatedAt string
	}
	rows := make([]row, 0)
	if err := r.db.WithContext(ctx).Raw(`
		SELECT id::text AS id,actor,action,metadata::text AS metadata,
			to_char(created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at
		FROM audit_events WHERE organization_id=? AND entity_type='project' AND entity_id=? ORDER BY created_at DESC`, organizationID, projectID).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("list project audit events: %w", err)
	}
	events := make([]AuditEvent, 0, len(rows))
	for _, item := range rows {
		metadata := make(map[string]any)
		if err := json.Unmarshal([]byte(item.Metadata), &metadata); err != nil {
			return nil, fmt.Errorf("decode project audit metadata: %w", err)
		}
		events = append(events, AuditEvent{ID: item.ID, Actor: item.Actor, Action: item.Action, Metadata: metadata, CreatedAt: item.CreatedAt})
	}
	return events, nil
}

func writeProjectAudit(tx *gorm.DB, organizationID, actor, action, projectID string, metadata map[string]any) error {
	return writeAudit(tx, organizationID, actor, action, "project", projectID, metadata)
}

func writeInvoiceAudit(tx *gorm.DB, organizationID, actor, invoiceID string, metadata map[string]any) error {
	return writeAudit(tx, organizationID, actor, "invoice.created_from_milestone", "invoice", invoiceID, metadata)
}

func writeAudit(tx *gorm.DB, organizationID, actor, action, entityType, entityID string, metadata map[string]any) error {
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("encode project audit metadata: %w", err)
	}
	if err := tx.Exec(`INSERT INTO audit_events (organization_id,actor,action,entity_type,entity_id,metadata) VALUES (?,?,?,?,?,?::jsonb)`, organizationID, actor, action, entityType, entityID, string(encoded)).Error; err != nil {
		return fmt.Errorf("write project audit event: %w", err)
	}
	return nil
}
