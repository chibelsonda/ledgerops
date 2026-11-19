package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/portfolio/ledgerops/backend/internal/platform/identity"
	"gorm.io/gorm"
)

type Repository interface {
	FindUserByEmail(context.Context, string) (*User, error)
	CreateAccount(context.Context, NewAccount) (*User, error)
	CreatePasswordReset(context.Context, string, []byte, time.Time, time.Time) (*PasswordResetRecipient, error)
	ConsumePasswordReset(context.Context, []byte, string, time.Time) error
	RecordFailedLogin(context.Context, string, time.Time, int, time.Duration) error
	ResetFailedLogins(context.Context, string) error
	CreateSession(context.Context, string, string, []byte, []byte, time.Time) error
	FindSession(context.Context, []byte, time.Time, time.Time) (*StoredSession, error)
	RotateCSRF(context.Context, int64, []byte, time.Time) error
	TouchSession(context.Context, int64, time.Time) error
	RevokeSession(context.Context, []byte, time.Time) error
}

func (r *PostgresRepository) CreateAccount(ctx context.Context, account NewAccount) (*User, error) {
	var created User
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var organizationID string
		if err := tx.Raw(`INSERT INTO organizations (name) VALUES (?) RETURNING id::text`, account.OrganizationName).Scan(&organizationID).Error; err != nil {
			return fmt.Errorf("create organization: %w", err)
		}
		var userID string
		if err := tx.Raw(`
			INSERT INTO users (email,display_name,password_hash)
			VALUES (?,?,?) RETURNING id::text`, account.Email, account.DisplayName, account.PasswordHash).Scan(&userID).Error; err != nil {
			var postgresError *pgconn.PgError
			if errors.As(err, &postgresError) && postgresError.Code == "23505" {
				return ErrEmailInUse
			}
			return fmt.Errorf("create user: %w", err)
		}
		if err := tx.Exec(`
			INSERT INTO organization_memberships (organization_id,user_id,role)
			VALUES (?,?,'admin')`, organizationID, userID).Error; err != nil {
			return fmt.Errorf("create organization membership: %w", err)
		}
		created = User{
			ID: userID, OrganizationID: organizationID, OrganizationName: account.OrganizationName,
			Email: account.Email, DisplayName: account.DisplayName, PasswordHash: account.PasswordHash,
			Role: identity.RoleAdmin, Active: true,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func (r *PostgresRepository) CreatePasswordReset(ctx context.Context, email string, tokenHash []byte, now, expiresAt time.Time) (*PasswordResetRecipient, error) {
	var recipient PasswordResetRecipient
	result := r.db.WithContext(ctx).Raw(`
		WITH target AS (
			SELECT id FROM users WHERE lower(email)=lower(?) AND active=true LIMIT 1
		), invalidated AS (
			UPDATE password_reset_tokens p SET used_at=? FROM target t
			WHERE p.user_id=t.id AND p.used_at IS NULL
		), inserted AS (
			INSERT INTO password_reset_tokens (user_id,token_hash,expires_at)
			SELECT id,?,? FROM target RETURNING user_id
		)
		SELECT u.email,u.display_name FROM inserted i JOIN users u ON u.id=i.user_id`,
		email, now, tokenHash, expiresAt).Scan(&recipient)
	if result.Error != nil {
		return nil, fmt.Errorf("create password reset: %w", result.Error)
	}
	if result.RowsAffected == 0 || recipient.Email == "" {
		return nil, nil
	}
	return &recipient, nil
}

func (r *PostgresRepository) ConsumePasswordReset(ctx context.Context, tokenHash []byte, passwordHash string, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var reset struct {
			ID     int64
			UserID string
		}
		result := tx.Raw(`
			SELECT id,user_id::text AS user_id FROM password_reset_tokens
			WHERE token_hash=? AND used_at IS NULL AND expires_at>?
			FOR UPDATE`, tokenHash, now).Scan(&reset)
		if result.Error != nil {
			return fmt.Errorf("find password reset: %w", result.Error)
		}
		if result.RowsAffected == 0 || reset.UserID == "" {
			return ErrInvalidResetToken
		}
		if err := tx.Exec(`
			UPDATE users SET password_hash=?,failed_login_attempts=0,locked_until=NULL,updated_at=?
			WHERE id=? AND active=true`, passwordHash, now, reset.UserID).Error; err != nil {
			return fmt.Errorf("update password: %w", err)
		}
		if err := tx.Exec(`UPDATE password_reset_tokens SET used_at=? WHERE id=?`, now, reset.ID).Error; err != nil {
			return fmt.Errorf("consume password reset: %w", err)
		}
		if err := tx.Exec(`UPDATE auth_sessions SET revoked_at=? WHERE user_id=? AND revoked_at IS NULL`, now, reset.UserID).Error; err != nil {
			return fmt.Errorf("revoke reset sessions: %w", err)
		}
		return nil
	})
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	result := r.db.WithContext(ctx).Raw(`
		SELECT u.id::text AS id, m.organization_id::text AS organization_id,
			o.name AS organization_name, u.email, u.display_name, u.password_hash,
			m.role, u.failed_login_attempts, u.locked_until, u.active
		FROM users u
		JOIN organization_memberships m ON m.user_id = u.id
		JOIN organizations o ON o.id = m.organization_id
		WHERE lower(u.email) = lower(?)
		ORDER BY m.created_at
		LIMIT 1`, email).Scan(&user)
	if result.Error != nil {
		return nil, fmt.Errorf("find user: %w", result.Error)
	}
	if result.RowsAffected == 0 || user.ID == "" {
		return nil, nil
	}
	return &user, nil
}

func (r *PostgresRepository) RecordFailedLogin(ctx context.Context, userID string, now time.Time, maxAttempts int, lockout time.Duration) error {
	lockedUntil := now.Add(lockout)
	if err := r.db.WithContext(ctx).Exec(`
		UPDATE users
		SET failed_login_attempts = CASE WHEN failed_login_attempts + 1 >= ? THEN 0 ELSE failed_login_attempts + 1 END,
			locked_until = CASE WHEN failed_login_attempts + 1 >= ? THEN ? ELSE NULL END,
			updated_at = now()
		WHERE id = ? AND active = true AND (locked_until IS NULL OR locked_until <= ?)`,
		maxAttempts, maxAttempts, lockedUntil, userID, now).Error; err != nil {
		return fmt.Errorf("record failed login: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ResetFailedLogins(ctx context.Context, userID string) error {
	if err := r.db.WithContext(ctx).Table("users").Where("id = ?", userID).Updates(map[string]any{
		"failed_login_attempts": 0,
		"locked_until":          nil,
		"updated_at":            gorm.Expr("now()"),
	}).Error; err != nil {
		return fmt.Errorf("reset failed logins: %w", err)
	}
	return nil
}

func (r *PostgresRepository) CreateSession(ctx context.Context, userID, organizationID string, tokenHash, csrfHash []byte, expiresAt time.Time) error {
	if err := r.db.WithContext(ctx).Exec(`
		INSERT INTO auth_sessions (user_id, organization_id, token_hash, csrf_hash, expires_at)
		VALUES (?, ?, ?, ?, ?)`, userID, organizationID, tokenHash, csrfHash, expiresAt).Error; err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (r *PostgresRepository) FindSession(ctx context.Context, tokenHash []byte, now, idleCutoff time.Time) (*StoredSession, error) {
	var row struct {
		ID               int64
		TokenHash        []byte
		CSRFHash         []byte
		ExpiresAt        time.Time
		LastSeenAt       time.Time
		UserID           string
		OrganizationID   string
		OrganizationName string
		Email            string
		DisplayName      string
		Role             identity.Role
	}
	result := r.db.WithContext(ctx).Raw(`
		SELECT s.id, s.token_hash, s.csrf_hash, s.expires_at, s.last_seen_at,
			u.id::text AS user_id, o.id::text AS organization_id,
			o.name AS organization_name, u.email, u.display_name, m.role
		FROM auth_sessions s
		JOIN users u ON u.id = s.user_id AND u.active = true
		JOIN organizations o ON o.id = s.organization_id
		JOIN organization_memberships m ON m.user_id = u.id AND m.organization_id = o.id
		WHERE s.token_hash = ? AND s.revoked_at IS NULL AND s.expires_at > ? AND s.last_seen_at > ?`, tokenHash, now, idleCutoff).Scan(&row)
	if result.Error != nil {
		return nil, fmt.Errorf("find session: %w", result.Error)
	}
	if result.RowsAffected == 0 || row.ID == 0 {
		return nil, nil
	}
	return &StoredSession{
		ID:         row.ID,
		TokenHash:  row.TokenHash,
		CSRFHash:   row.CSRFHash,
		ExpiresAt:  row.ExpiresAt,
		LastSeenAt: row.LastSeenAt,
		Principal: identity.Principal{
			UserID:           row.UserID,
			OrganizationID:   row.OrganizationID,
			OrganizationName: row.OrganizationName,
			Email:            row.Email,
			DisplayName:      row.DisplayName,
			Role:             row.Role,
		},
	}, nil
}

func (r *PostgresRepository) TouchSession(ctx context.Context, sessionID int64, now time.Time) error {
	if err := r.db.WithContext(ctx).Table("auth_sessions").Where("id = ? AND revoked_at IS NULL", sessionID).Update("last_seen_at", now).Error; err != nil {
		return fmt.Errorf("touch session: %w", err)
	}
	return nil
}

func (r *PostgresRepository) RotateCSRF(ctx context.Context, sessionID int64, csrfHash []byte, now time.Time) error {
	if err := r.db.WithContext(ctx).Table("auth_sessions").Where("id = ?", sessionID).Updates(map[string]any{
		"csrf_hash":    csrfHash,
		"last_seen_at": now,
	}).Error; err != nil {
		return fmt.Errorf("rotate CSRF token: %w", err)
	}
	return nil
}

func (r *PostgresRepository) RevokeSession(ctx context.Context, tokenHash []byte, now time.Time) error {
	if err := r.db.WithContext(ctx).Table("auth_sessions").Where("token_hash = ? AND revoked_at IS NULL", tokenHash).Update("revoked_at", now).Error; err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}
