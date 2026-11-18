package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/portfolio/ledgerops/backend/internal/platform/identity"
	"gorm.io/gorm"
)

type Repository interface {
	FindUserByEmail(context.Context, string) (*User, error)
	RecordFailedLogin(context.Context, string, int, *time.Time) error
	ResetFailedLogins(context.Context, string) error
	CreateSession(context.Context, string, string, []byte, []byte, time.Time) error
	FindSession(context.Context, []byte, time.Time) (*StoredSession, error)
	RotateCSRF(context.Context, int64, []byte, time.Time) error
	RevokeSession(context.Context, []byte, time.Time) error
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

func (r *PostgresRepository) RecordFailedLogin(ctx context.Context, userID string, attempts int, lockedUntil *time.Time) error {
	if err := r.db.WithContext(ctx).Table("users").Where("id = ?", userID).Updates(map[string]any{
		"failed_login_attempts": attempts,
		"locked_until":          lockedUntil,
		"updated_at":            gorm.Expr("now()"),
	}).Error; err != nil {
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

func (r *PostgresRepository) FindSession(ctx context.Context, tokenHash []byte, now time.Time) (*StoredSession, error) {
	var row struct {
		ID               int64
		TokenHash        []byte
		CSRFHash         []byte
		ExpiresAt        time.Time
		UserID           string
		OrganizationID   string
		OrganizationName string
		Email            string
		DisplayName      string
		Role             identity.Role
	}
	result := r.db.WithContext(ctx).Raw(`
		SELECT s.id, s.token_hash, s.csrf_hash, s.expires_at,
			u.id::text AS user_id, o.id::text AS organization_id,
			o.name AS organization_name, u.email, u.display_name, m.role
		FROM auth_sessions s
		JOIN users u ON u.id = s.user_id AND u.active = true
		JOIN organizations o ON o.id = s.organization_id
		JOIN organization_memberships m ON m.user_id = u.id AND m.organization_id = o.id
		WHERE s.token_hash = ? AND s.revoked_at IS NULL AND s.expires_at > ?`, tokenHash, now).Scan(&row)
	if result.Error != nil {
		return nil, fmt.Errorf("find session: %w", result.Error)
	}
	if result.RowsAffected == 0 || row.ID == 0 {
		return nil, nil
	}
	return &StoredSession{
		ID:        row.ID,
		TokenHash: row.TokenHash,
		CSRFHash:  row.CSRFHash,
		ExpiresAt: row.ExpiresAt,
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
