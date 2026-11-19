package clients

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

const clientProjection = `c.id::text AS id, c.name, c.payment_terms_days, c.reminders_enabled, c.reminder_delay_days,
	COALESCE((SELECT name FROM client_contacts WHERE client_id = c.id AND is_primary LIMIT 1), '') AS primary_contact,
	COALESCE((SELECT email FROM client_contacts WHERE client_id = c.id AND is_primary LIMIT 1), '') AS primary_email,
	(SELECT COUNT(*)::int FROM client_contacts WHERE client_id = c.id) AS contact_count,
	(SELECT COUNT(*)::int FROM invoices WHERE client_id = c.id) AS invoice_count,
	COALESCE((SELECT SUM(b.outstanding_minor)
	 FROM invoices i JOIN invoice_balances b ON b.invoice_id=i.id AND b.organization_id=i.organization_id
	 WHERE i.client_id=c.id AND i.status IN ('open','partial','overdue')),0)::bigint AS outstanding_minor,
	COALESCE((SELECT base_currency FROM organizations WHERE id=c.organization_id),'USD') AS currency_code,
	to_char(c.created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at,
	to_char(c.updated_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') AS updated_at`

func (r *PostgresRepository) List(ctx context.Context, organizationID string, params ListParams) (Page, error) {
	base := r.db.WithContext(ctx).Table("clients c").Where("c.organization_id = ? AND c.source_deleted_at IS NULL", organizationID)
	if params.Query != "" {
		like := "%" + params.Query + "%"
		base = base.Where(`c.name ILIKE ? OR EXISTS(SELECT 1 FROM client_contacts cc WHERE cc.client_id = c.id AND (cc.name ILIKE ? OR cc.email ILIKE ?))`, like, like, like)
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return Page{}, fmt.Errorf("count clients: %w", err)
	}
	items := make([]Client, 0)
	if err := base.Select(clientProjection).
		Order("c.name").Limit(params.PageSize).Offset((params.Page - 1) * params.PageSize).Scan(&items).Error; err != nil {
		return Page{}, fmt.Errorf("list clients: %w", err)
	}
	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(params.PageSize) - 1) / int64(params.PageSize))
	}
	return Page{Items: items, Pagination: Pagination{Page: params.Page, PageSize: params.PageSize, TotalItems: total, TotalPages: totalPages}}, nil
}

func (r *PostgresRepository) Create(ctx context.Context, organizationID string, actor Actor, input MutationInput) (Client, error) {
	var clientID string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row := struct{ ID string }{}
		if err := tx.Raw(`INSERT INTO clients (organization_id,name,payment_terms_days,reminders_enabled,reminder_delay_days) VALUES (?,?,?,?,?) RETURNING id::text AS id`, organizationID, input.Name, input.PaymentTermsDays, input.RemindersEnabled, input.ReminderDelayDays).Scan(&row).Error; err != nil {
			return fmt.Errorf("insert client: %w", err)
		}
		clientID = row.ID
		contact := input.PrimaryContact
		if err := tx.Exec(`INSERT INTO client_contacts (organization_id,client_id,name,email,phone,role_title,is_primary) VALUES (?,?,?,?,NULLIF(?,''),NULLIF(?,''),true)`, organizationID, clientID, contact.Name, contact.Email, contact.Phone, contact.RoleTitle).Error; err != nil {
			return fmt.Errorf("insert primary contact: %w", err)
		}
		if err := enqueueQuickBooksWrite(tx, organizationID, "client", clientID, "create"); err != nil {
			return err
		}
		return clientAudit(tx, organizationID, actor.Email, "client.created", clientID, map[string]any{"name": input.Name})
	})
	if err != nil {
		return Client{}, fmt.Errorf("create client transaction: %w", err)
	}
	return r.get(ctx, organizationID, clientID)
}

func (r *PostgresRepository) Update(ctx context.Context, organizationID, clientID string, actor Actor, input MutationInput) (Client, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Table("clients").Where("id=? AND organization_id=?", clientID, organizationID).Updates(map[string]any{
			"name": input.Name, "payment_terms_days": input.PaymentTermsDays, "reminders_enabled": input.RemindersEnabled,
			"reminder_delay_days": input.ReminderDelayDays, "updated_at": gorm.Expr("now()"),
		})
		if result.Error != nil {
			return fmt.Errorf("update client: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return apperrors.ErrNotFound
		}
		if err := enqueueQuickBooksWrite(tx, organizationID, "client", clientID, "update"); err != nil {
			return err
		}
		return clientAudit(tx, organizationID, actor.Email, "client.updated", clientID, map[string]any{"note": input.AuditNote})
	})
	if err != nil {
		return Client{}, fmt.Errorf("update client transaction: %w", err)
	}
	return r.get(ctx, organizationID, clientID)
}

func (r *PostgresRepository) ListContacts(ctx context.Context, organizationID, clientID string) ([]Contact, error) {
	var exists bool
	if err := r.db.WithContext(ctx).Raw(`SELECT EXISTS(SELECT 1 FROM clients WHERE id=? AND organization_id=?)`, clientID, organizationID).Scan(&exists).Error; err != nil {
		return nil, fmt.Errorf("check client: %w", err)
	}
	if !exists {
		return nil, apperrors.ErrNotFound
	}
	items := make([]Contact, 0)
	if err := r.db.WithContext(ctx).Raw(`SELECT id::text AS id,client_id::text AS client_id,name,email,COALESCE(phone,'') AS phone,COALESCE(role_title,'') AS role_title,is_primary,to_char(created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at,to_char(updated_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') AS updated_at FROM client_contacts WHERE organization_id=? AND client_id=? ORDER BY is_primary DESC,name`, organizationID, clientID).Scan(&items).Error; err != nil {
		return nil, fmt.Errorf("list contacts: %w", err)
	}
	return items, nil
}

func (r *PostgresRepository) CreateContact(ctx context.Context, organizationID, clientID string, actor Actor, input ContactInput) (Contact, error) {
	var contactID string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var exists bool
		if err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM clients WHERE id=? AND organization_id=?)`, clientID, organizationID).Scan(&exists).Error; err != nil || !exists {
			if err != nil {
				return err
			}
			return apperrors.ErrNotFound
		}
		if input.IsPrimary {
			if err := tx.Table("client_contacts").Where("client_id=?", clientID).Update("is_primary", false).Error; err != nil {
				return err
			}
		}
		row := struct{ ID string }{}
		result := tx.Raw(`INSERT INTO client_contacts (organization_id,client_id,name,email,phone,role_title,is_primary) VALUES (?,?,?,?,NULLIF(?,''),NULLIF(?,''),?) RETURNING id::text AS id`, organizationID, clientID, input.Name, input.Email, input.Phone, input.RoleTitle, input.IsPrimary).Scan(&row)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
				return apperrors.ErrConflict
			}
			return result.Error
		}
		contactID = row.ID
		return clientAudit(tx, organizationID, actor.Email, "contact.created", clientID, map[string]any{"contactId": contactID, "email": input.Email})
	})
	if err != nil {
		return Contact{}, fmt.Errorf("create contact transaction: %w", err)
	}
	return r.getContact(ctx, organizationID, contactID)
}

func (r *PostgresRepository) UpdateContact(ctx context.Context, organizationID, contactID string, actor Actor, input ContactInput) (Contact, error) {
	var clientID string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Raw(`SELECT client_id::text FROM client_contacts WHERE id=? AND organization_id=? FOR UPDATE`, contactID, organizationID).Scan(&clientID).Error; err != nil {
			return err
		}
		if clientID == "" {
			return apperrors.ErrNotFound
		}
		if input.IsPrimary {
			if err := tx.Table("client_contacts").Where("client_id=? AND id<>?", clientID, contactID).Update("is_primary", false).Error; err != nil {
				return err
			}
		}
		result := tx.Table("client_contacts").Where("id=? AND organization_id=?", contactID, organizationID).Updates(map[string]any{"name": input.Name, "email": input.Email, "phone": input.Phone, "role_title": input.RoleTitle, "is_primary": input.IsPrimary, "updated_at": gorm.Expr("now()")})
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
				return apperrors.ErrConflict
			}
			return result.Error
		}
		return clientAudit(tx, organizationID, actor.Email, "contact.updated", clientID, map[string]any{"contactId": contactID})
	})
	if err != nil {
		return Contact{}, fmt.Errorf("update contact transaction: %w", err)
	}
	return r.getContact(ctx, organizationID, contactID)
}

func (r *PostgresRepository) get(ctx context.Context, organizationID, clientID string) (Client, error) {
	var result Client
	if err := r.db.WithContext(ctx).Table("clients c").Select(clientProjection).Where("c.id=? AND c.organization_id=?", clientID, organizationID).Scan(&result).Error; err != nil {
		return result, fmt.Errorf("get client: %w", err)
	}
	if result.ID == "" {
		return result, apperrors.ErrNotFound
	}
	return result, nil
}

func (r *PostgresRepository) getContact(ctx context.Context, organizationID, contactID string) (Contact, error) {
	var result Contact
	if err := r.db.WithContext(ctx).Raw(`SELECT id::text AS id,client_id::text AS client_id,name,email,COALESCE(phone,'') AS phone,COALESCE(role_title,'') AS role_title,is_primary,to_char(created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at,to_char(updated_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') AS updated_at FROM client_contacts WHERE id=? AND organization_id=?`, contactID, organizationID).Scan(&result).Error; err != nil {
		return result, err
	}
	if result.ID == "" {
		return result, apperrors.ErrNotFound
	}
	return result, nil
}

func clientAudit(tx *gorm.DB, organizationID, actor, action, clientID string, metadata map[string]any) error {
	return audit.Write(tx, organizationID, actor, action, "client", clientID, metadata)
}

func enqueueQuickBooksWrite(tx *gorm.DB, organizationID, aggregateType, aggregateID, operation string) error {
	if err := tx.Exec(`INSERT INTO quickbooks_outbox (organization_id, aggregate_type, aggregate_id, operation, idempotency_key)
		SELECT ?, ?, ?::uuid, ?, concat(?::text, ':', ?::text, ':', ?::uuid::text, ':', txid_current()::text)
		WHERE EXISTS (SELECT 1 FROM quickbooks_connections WHERE organization_id = ?)
		AND (? <> 'client' OR EXISTS (SELECT 1 FROM clients WHERE id = ? AND (external_id IS NULL OR external_id LIKE 'quickbooks:customer:%')))`,
		organizationID, aggregateType, aggregateID, operation, organizationID, operation, aggregateID, organizationID,
		aggregateType, aggregateID).Error; err != nil {
		return fmt.Errorf("enqueue QuickBooks %s %s: %w", aggregateType, operation, err)
	}
	return nil
}
