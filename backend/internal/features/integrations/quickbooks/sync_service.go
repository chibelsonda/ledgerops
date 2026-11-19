package quickbooks

import (
	"context"
	"errors"
	"fmt"
	"time"

	ledgersync "github.com/portfolio/ledgerops/backend/internal/features/sync"
)

const (
	syncPageSize       = 100
	tokenRefreshLeeway = 5 * time.Minute
)

var ErrConnectionMissing = errors.New("QuickBooks connection is missing")

type PermanentError struct{ Err error }

func (e *PermanentError) Error() string             { return e.Err.Error() }
func (e *PermanentError) Unwrap() error             { return e.Err }
func (e *PermanentError) Retryable() bool           { return false }
func (e *PermanentError) RetryAfter() time.Duration { return 0 }

type SyncService struct {
	repository SyncRepository
	client     AccountingClient
	cipher     Cipher
	now        func() time.Time
}

type syncState struct {
	connection   *Connection
	accessToken  string
	refreshToken string
}

func NewSyncService(repository SyncRepository, client AccountingClient, cipher Cipher) *SyncService {
	return &SyncService{repository: repository, client: client, cipher: cipher, now: time.Now}
}

func (s *SyncService) Synchronize(ctx context.Context, job ledgersync.Job) (int, error) {
	organizationID := job.OrganizationID
	connection, err := s.repository.FindConnection(ctx, organizationID)
	if err != nil {
		return 0, err
	}
	if connection == nil {
		return 0, &PermanentError{Err: ErrConnectionMissing}
	}
	accessToken, err := s.cipher.Decrypt(connection.AccessToken)
	if err != nil {
		return 0, &PermanentError{Err: fmt.Errorf("decrypt QuickBooks access token: %w", err)}
	}
	refreshToken, err := s.cipher.Decrypt(connection.RefreshToken)
	if err != nil {
		return 0, &PermanentError{Err: fmt.Errorf("decrypt QuickBooks refresh token: %w", err)}
	}
	state := &syncState{connection: connection, accessToken: accessToken, refreshToken: refreshToken}
	if !connection.AccessTokenExpiresAt.After(s.now().Add(tokenRefreshLeeway)) {
		if err := s.refresh(ctx, organizationID, state); err != nil {
			return 0, err
		}
	}

	cursors := make(map[string]time.Time, len(ImportEntityTypes))
	allIncremental := true
	var earliest time.Time
	for _, entityType := range ImportEntityTypes {
		cursor, cursorErr := s.repository.Cursor(ctx, organizationID, entityType)
		if cursorErr != nil {
			return 0, cursorErr
		}
		if cursor == nil || cursor.Before(s.now().Add(-29*24*time.Hour)) {
			allIncremental = false
			continue
		}
		cursors[entityType] = *cursor
		if earliest.IsZero() || cursor.Before(earliest) {
			earliest = *cursor
		}
	}
	if allIncremental {
		return s.synchronizeIncremental(ctx, job.SyncRunID, state, earliest)
	}
	return s.synchronizeFull(ctx, job.SyncRunID, state)
}

func (s *SyncService) synchronizeFull(ctx context.Context, syncRunID string, state *syncState) (int, error) {
	organizationID := state.connection.OrganizationID
	cursorAt := s.now().UTC()
	total := 0
	for _, entityType := range ImportEntityTypes {
		if err := s.repository.UpdateEntityProgress(ctx, syncRunID, entityType, "running", 0, ""); err != nil {
			return total, err
		}
		count, err := syncPages(ctx, s, organizationID, state,
			func(ctx context.Context, token, realmID string, start, maximum int) (Page[RemoteEntity], error) {
				return s.client.ListRemoteEntities(ctx, token, realmID, entityType, start, maximum)
			},
			s.repository.UpsertRemoteEntities,
		)
		if err != nil {
			_ = s.repository.UpdateEntityProgress(ctx, syncRunID, entityType, "failed", count, err.Error())
			return total + count, err
		}
		missing, err := s.repository.ReconcileFullSnapshot(ctx, organizationID, entityType, cursorAt)
		if err != nil {
			_ = s.repository.UpdateEntityProgress(ctx, syncRunID, entityType, "failed", count, err.Error())
			return total + count, err
		}
		if err := s.syncCoreProjection(ctx, organizationID, state, entityType); err != nil {
			_ = s.repository.UpdateEntityProgress(ctx, syncRunID, entityType, "failed", count, err.Error())
			return total + count, err
		}
		if err := s.repository.ReconcileDeletedEntities(ctx, organizationID, missing); err != nil {
			_ = s.repository.UpdateEntityProgress(ctx, syncRunID, entityType, "failed", count, err.Error())
			return total + count, err
		}
		if err := s.repository.SaveCursor(ctx, organizationID, entityType, cursorAt, true); err != nil {
			return total + count, err
		}
		if err := s.repository.UpdateEntityProgress(ctx, syncRunID, entityType, "succeeded", count, ""); err != nil {
			return total + count, err
		}
		total += count
	}
	return total, nil
}

func (s *SyncService) synchronizeIncremental(ctx context.Context, syncRunID string, state *syncState, changedSince time.Time) (int, error) {
	organizationID := state.connection.OrganizationID
	for _, entityType := range ImportEntityTypes {
		if err := s.repository.UpdateEntityProgress(ctx, syncRunID, entityType, "running", 0, ""); err != nil {
			return 0, err
		}
	}
	result, err := s.client.ChangeDataCapture(ctx, state.accessToken, state.connection.RealmID, ImportEntityTypes, changedSince)
	if isUnauthorized(err) {
		if refreshErr := s.refresh(ctx, organizationID, state); refreshErr != nil {
			return 0, refreshErr
		}
		result, err = s.client.ChangeDataCapture(ctx, state.accessToken, state.connection.RealmID, ImportEntityTypes, changedSince)
	}
	if err != nil {
		for _, entityType := range ImportEntityTypes {
			_ = s.repository.UpdateEntityProgress(ctx, syncRunID, entityType, "failed", 0, err.Error())
		}
		return 0, err
	}
	if err := s.repository.UpsertRemoteEntities(ctx, organizationID, result.Entities); err != nil {
		return 0, err
	}
	if err := s.repository.ReconcileDeletedEntities(ctx, organizationID, result.Entities); err != nil {
		return 0, err
	}
	counts := make(map[string]int, len(ImportEntityTypes))
	changed := make(map[string]bool, len(ImportEntityTypes))
	for _, record := range result.Entities {
		counts[record.EntityType]++
		if record.DeletedAt == nil {
			changed[record.EntityType] = true
		}
	}
	for _, entityType := range ImportEntityTypes {
		if changed[entityType] {
			if err := s.syncCoreProjection(ctx, organizationID, state, entityType); err != nil {
				_ = s.repository.UpdateEntityProgress(ctx, syncRunID, entityType, "failed", counts[entityType], err.Error())
				return len(result.Entities), err
			}
		}
		if err := s.repository.SaveCursor(ctx, organizationID, entityType, result.CursorAt, false); err != nil {
			return len(result.Entities), err
		}
		if err := s.repository.UpdateEntityProgress(ctx, syncRunID, entityType, "succeeded", counts[entityType], ""); err != nil {
			return len(result.Entities), err
		}
	}
	if err := s.repository.MarkWebhookEventsProcessed(ctx, organizationID, result.CursorAt); err != nil {
		return len(result.Entities), err
	}
	return len(result.Entities), nil
}

func (s *SyncService) syncCoreProjection(ctx context.Context, organizationID string, state *syncState, entityType string) error {
	switch entityType {
	case EntityCustomer:
		_, err := syncPages(ctx, s, organizationID, state, s.client.ListCustomers, s.repository.UpsertCustomers)
		return err
	case EntityInvoice:
		_, err := syncPages(ctx, s, organizationID, state, s.client.ListInvoices, s.repository.UpsertInvoices)
		return err
	case EntityPayment:
		_, err := syncPages(ctx, s, organizationID, state, s.client.ListPayments, s.repository.InsertPayments)
		return err
	case EntityPurchase:
		_, err := syncPages(ctx, s, organizationID, state, s.client.ListExpenses, s.repository.UpsertExpenses)
		return err
	default:
		return nil
	}
}

func syncPages[T any](
	ctx context.Context,
	service *SyncService,
	organizationID string,
	state *syncState,
	fetch func(context.Context, string, string, int, int) (Page[T], error),
	persist func(context.Context, string, []T) error,
) (int, error) {
	total := 0
	for start := 1; ; start += syncPageSize {
		page, err := fetch(ctx, state.accessToken, state.connection.RealmID, start, syncPageSize)
		if isUnauthorized(err) {
			if refreshErr := service.refresh(ctx, organizationID, state); refreshErr != nil {
				return total, refreshErr
			}
			page, err = fetch(ctx, state.accessToken, state.connection.RealmID, start, syncPageSize)
		}
		if err != nil {
			return total, err
		}
		if len(page.Items) > 0 {
			if err := persist(ctx, organizationID, page.Items); err != nil {
				return total, err
			}
			total += len(page.Items)
		}
		if !page.HasMore {
			return total, nil
		}
	}
}

func (s *SyncService) refresh(ctx context.Context, organizationID string, state *syncState) error {
	if !state.connection.RefreshTokenExpiresAt.After(s.now()) {
		return &PermanentError{Err: fmt.Errorf("QuickBooks authorization expired; reconnect the company")}
	}
	tokens, err := s.client.RefreshTokens(ctx, state.refreshToken)
	if err != nil {
		return err
	}
	accessCiphertext, err := s.cipher.Encrypt(tokens.AccessToken)
	if err != nil {
		return &PermanentError{Err: fmt.Errorf("encrypt refreshed QuickBooks access token: %w", err)}
	}
	refreshCiphertext, err := s.cipher.Encrypt(tokens.RefreshToken)
	if err != nil {
		return &PermanentError{Err: fmt.Errorf("encrypt refreshed QuickBooks refresh token: %w", err)}
	}
	now := s.now()
	if err := s.repository.UpdateTokens(ctx, organizationID, accessCiphertext, refreshCiphertext,
		now.Add(tokens.AccessTokenExpiresIn), now.Add(tokens.RefreshTokenExpiresIn), tokens.Scope); err != nil {
		return err
	}
	state.accessToken = tokens.AccessToken
	state.refreshToken = tokens.RefreshToken
	state.connection.AccessToken = accessCiphertext
	state.connection.RefreshToken = refreshCiphertext
	state.connection.AccessTokenExpiresAt = now.Add(tokens.AccessTokenExpiresIn)
	state.connection.RefreshTokenExpiresAt = now.Add(tokens.RefreshTokenExpiresIn)
	return nil
}
