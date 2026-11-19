package quickbooks

import (
	"context"
	"testing"
	"time"

	ledgersync "github.com/portfolio/ledgerops/backend/internal/features/sync"
)

type syncRepositoryStub struct {
	connection    *Connection
	tokensUpdated bool
	operations    []string
}

func (r *syncRepositoryStub) FindConnection(context.Context, string) (*Connection, error) {
	return r.connection, nil
}
func (r *syncRepositoryStub) UpdateTokens(_ context.Context, _ string, access, refresh []byte, accessExpiry, refreshExpiry time.Time, _ string) error {
	r.tokensUpdated = true
	r.connection.AccessToken = access
	r.connection.RefreshToken = refresh
	r.connection.AccessTokenExpiresAt = accessExpiry
	r.connection.RefreshTokenExpiresAt = refreshExpiry
	return nil
}
func (r *syncRepositoryStub) UpsertCustomers(_ context.Context, _ string, records []CustomerRecord) error {
	r.operations = append(r.operations, "customers")
	return nil
}
func (r *syncRepositoryStub) UpsertInvoices(_ context.Context, _ string, records []InvoiceRecord) error {
	r.operations = append(r.operations, "invoices")
	return nil
}
func (r *syncRepositoryStub) InsertPayments(_ context.Context, _ string, records []PaymentRecord) error {
	r.operations = append(r.operations, "payments")
	return nil
}
func (r *syncRepositoryStub) UpsertExpenses(_ context.Context, _ string, records []ExpenseRecord) error {
	r.operations = append(r.operations, "expenses")
	return nil
}
func (r *syncRepositoryStub) UpsertRemoteEntities(context.Context, string, []RemoteEntity) error {
	return nil
}
func (r *syncRepositoryStub) ReconcileDeletedEntities(context.Context, string, []RemoteEntity) error {
	return nil
}
func (r *syncRepositoryStub) ReconcileFullSnapshot(context.Context, string, string, time.Time) ([]RemoteEntity, error) {
	return nil, nil
}
func (r *syncRepositoryStub) Cursor(context.Context, string, string) (*time.Time, error) {
	return nil, nil
}
func (r *syncRepositoryStub) SaveCursor(context.Context, string, string, time.Time, bool) error {
	return nil
}
func (r *syncRepositoryStub) UpdateEntityProgress(context.Context, string, string, string, int, string) error {
	return nil
}
func (r *syncRepositoryStub) MarkWebhookEventsProcessed(context.Context, string, time.Time) error {
	return nil
}

type accountingClientStub struct {
	accessTokens []string
}

func (c *accountingClientStub) RefreshTokens(context.Context, string) (Tokens, error) {
	return Tokens{AccessToken: "new-access", RefreshToken: "new-refresh", AccessTokenExpiresIn: time.Hour, RefreshTokenExpiresIn: 100 * 24 * time.Hour}, nil
}
func (c *accountingClientStub) RevokeToken(context.Context, string) error { return nil }
func (c *accountingClientStub) record(token string)                       { c.accessTokens = append(c.accessTokens, token) }
func (c *accountingClientStub) ListCustomers(_ context.Context, token, _ string, _, _ int) (Page[CustomerRecord], error) {
	c.record(token)
	return Page[CustomerRecord]{Items: []CustomerRecord{{ExternalID: "customer"}}}, nil
}
func (c *accountingClientStub) ListInvoices(_ context.Context, token, _ string, _, _ int) (Page[InvoiceRecord], error) {
	c.record(token)
	return Page[InvoiceRecord]{Items: []InvoiceRecord{{ExternalID: "invoice"}}}, nil
}
func (c *accountingClientStub) ListPayments(_ context.Context, token, _ string, _, _ int) (Page[PaymentRecord], error) {
	c.record(token)
	return Page[PaymentRecord]{Items: []PaymentRecord{{ExternalID: "payment"}}}, nil
}
func (c *accountingClientStub) ListExpenses(_ context.Context, token, _ string, _, _ int) (Page[ExpenseRecord], error) {
	c.record(token)
	return Page[ExpenseRecord]{Items: []ExpenseRecord{{ExternalID: "expense"}}}, nil
}
func (c *accountingClientStub) ListRemoteEntities(_ context.Context, token, _ string, entityType string, _, _ int) (Page[RemoteEntity], error) {
	c.record(token)
	return Page[RemoteEntity]{Items: []RemoteEntity{{EntityType: entityType, ExternalID: "remote"}}}, nil
}
func (c *accountingClientStub) ChangeDataCapture(context.Context, string, string, []string, time.Time) (CDCResult, error) {
	return CDCResult{}, nil
}

func TestSynchronizeRefreshesAndPersistsEntitiesInDependencyOrder(t *testing.T) {
	now := time.Date(2026, time.November, 19, 9, 0, 0, 0, time.UTC)
	repository := &syncRepositoryStub{connection: &Connection{
		OrganizationID: "organization", RealmID: "123456", AccessToken: []byte("encrypted:old-access"), RefreshToken: []byte("encrypted:old-refresh"),
		AccessTokenExpiresAt: now.Add(time.Minute), RefreshTokenExpiresAt: now.Add(30 * 24 * time.Hour),
	}}
	client := &accountingClientStub{}
	service := NewSyncService(repository, client, prefixCipher{})
	service.now = func() time.Time { return now }

	processed, err := service.Synchronize(context.Background(), ledgersync.Job{OrganizationID: "organization", SyncRunID: "run"})
	if err != nil {
		t.Fatal(err)
	}
	if processed != len(ImportEntityTypes) || !repository.tokensUpdated {
		t.Fatalf("processed = %d, tokens updated = %v", processed, repository.tokensUpdated)
	}
	wantOrder := []string{"customers", "invoices", "payments", "expenses"}
	for index, operation := range wantOrder {
		if repository.operations[index] != operation {
			t.Fatalf("operations = %v", repository.operations)
		}
		if client.accessTokens[index] != "new-access" {
			t.Fatalf("access tokens = %v", client.accessTokens)
		}
	}
}
