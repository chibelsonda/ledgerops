package quickbooks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

var ErrInvalidState = errors.New("invalid or expired OAuth state")

type Repository interface {
	FindConnection(context.Context, string) (*Connection, error)
	SaveState(context.Context, string, []byte, time.Time) error
	ConsumeState(context.Context, []byte, time.Time) (string, error)
	UpsertConnection(context.Context, Connection) error
	DeleteConnection(context.Context, string) error
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) FindConnection(ctx context.Context, organizationID string) (*Connection, error) {
	var connection Connection
	result := r.db.WithContext(ctx).Raw(`
		SELECT organization_id::text AS organization_id, realm_id, company_name,
			access_token_ciphertext AS access_token, refresh_token_ciphertext AS refresh_token,
			access_token_expires_at, refresh_token_expires_at, scope, connected_at
		FROM quickbooks_connections WHERE organization_id = ?`, organizationID).Scan(&connection)
	if result.Error != nil {
		return nil, fmt.Errorf("find QuickBooks connection: %w", result.Error)
	}
	if result.RowsAffected == 0 || connection.OrganizationID == "" {
		return nil, nil
	}
	return &connection, nil
}

func (r *PostgresRepository) SaveState(ctx context.Context, organizationID string, stateHash []byte, expiresAt time.Time) error {
	if err := r.db.WithContext(ctx).Exec(`
		INSERT INTO quickbooks_oauth_states (organization_id, state_hash, expires_at)
		VALUES (?, ?, ?)`, organizationID, stateHash, expiresAt).Error; err != nil {
		return fmt.Errorf("save QuickBooks OAuth state: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ConsumeState(ctx context.Context, stateHash []byte, now time.Time) (string, error) {
	var organizationID string
	result := r.db.WithContext(ctx).Raw(`
		UPDATE quickbooks_oauth_states
		SET consumed_at = ?
		WHERE state_hash = ? AND consumed_at IS NULL AND expires_at > ?
		RETURNING organization_id::text`, now, stateHash, now).Scan(&organizationID)
	if result.Error != nil {
		return "", fmt.Errorf("consume QuickBooks OAuth state: %w", result.Error)
	}
	if result.RowsAffected == 0 || organizationID == "" {
		return "", ErrInvalidState
	}
	return organizationID, nil
}

func (r *PostgresRepository) UpsertConnection(ctx context.Context, connection Connection) error {
	if err := r.db.WithContext(ctx).Exec(`
		INSERT INTO quickbooks_connections (
			organization_id, realm_id, company_name, access_token_ciphertext,
			refresh_token_ciphertext, access_token_expires_at,
			refresh_token_expires_at, scope
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (organization_id) DO UPDATE SET
			realm_id = EXCLUDED.realm_id,
			company_name = EXCLUDED.company_name,
			access_token_ciphertext = EXCLUDED.access_token_ciphertext,
			refresh_token_ciphertext = EXCLUDED.refresh_token_ciphertext,
			access_token_expires_at = EXCLUDED.access_token_expires_at,
			refresh_token_expires_at = EXCLUDED.refresh_token_expires_at,
			scope = EXCLUDED.scope,
			connected_at = now(),
			updated_at = now()`,
		connection.OrganizationID, connection.RealmID, connection.CompanyName,
		connection.AccessToken, connection.RefreshToken, connection.AccessTokenExpiresAt,
		connection.RefreshTokenExpiresAt, connection.Scope,
	).Error; err != nil {
		return fmt.Errorf("save QuickBooks connection: %w", err)
	}
	return nil
}

func (r *PostgresRepository) DeleteConnection(ctx context.Context, organizationID string) error {
	if err := r.db.WithContext(ctx).Exec(
		"DELETE FROM quickbooks_connections WHERE organization_id = ?", organizationID,
	).Error; err != nil {
		return fmt.Errorf("delete QuickBooks connection: %w", err)
	}
	return nil
}
