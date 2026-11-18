//go:build integration

package invoices

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/portfolio/ledgerops/backend/internal/platform/apperrors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestPostgresRepositoryIsolatesInvoiceTenants(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is required for tenant-isolation integration tests")
	}

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin test transaction: %v", tx.Error)
	}
	t.Cleanup(func() { _ = tx.Rollback().Error })

	marker := fmt.Sprintf("tenant-isolation-%d", time.Now().UnixNano())
	organizationA := insertTestOrganization(t, tx, marker+"-a")
	organizationB := insertTestOrganization(t, tx, marker+"-b")
	clientA := insertTestClient(t, tx, organizationA, marker+"-client-a")
	clientB := insertTestClient(t, tx, organizationB, marker+"-client-b")
	invoiceA := insertTestInvoice(t, tx, organizationA, clientA, marker+"-invoice-a", 10_000)
	invoiceB := insertTestInvoice(t, tx, organizationB, clientB, marker+"-invoice-b", 20_000)

	repository := NewPostgresRepository(tx)
	page, err := repository.List(context.Background(), organizationA, ListParams{Query: marker, Page: 1, PageSize: 50})
	if err != nil {
		t.Fatalf("list tenant A invoices: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != invoiceA {
		t.Fatalf("tenant A invoice list = %#v, want only %s", page.Items, invoiceA)
	}

	_, err = repository.Update(context.Background(), organizationA, invoiceB, "tenant-test@ledgerops.invalid", MutationInput{
		ClientID: clientA, InvoiceNumber: marker + "-blocked", Status: "open", CurrencyCode: "USD",
		AmountMinor: 99_000, IssuedOn: "2026-11-01", DueOn: "2026-11-30", AuditNote: "Cross-tenant update must be rejected.",
	})
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Fatalf("cross-tenant update error = %v, want not found", err)
	}

	var amountMinor int64
	if err := tx.Raw("SELECT amount_minor FROM invoices WHERE id = ? AND organization_id = ?", invoiceB, organizationB).Scan(&amountMinor).Error; err != nil {
		t.Fatalf("verify tenant B invoice: %v", err)
	}
	if amountMinor != 20_000 {
		t.Fatalf("tenant B invoice amount = %d, want 20000", amountMinor)
	}
}

func insertTestOrganization(t *testing.T, tx *gorm.DB, name string) string {
	t.Helper()
	return insertTestID(t, tx, "INSERT INTO organizations (name, base_currency) VALUES (?, 'USD') RETURNING id::text AS id", name)
}

func insertTestClient(t *testing.T, tx *gorm.DB, organizationID, name string) string {
	t.Helper()
	return insertTestID(t, tx, "INSERT INTO clients (organization_id, name) VALUES (?, ?) RETURNING id::text AS id", organizationID, name)
}

func insertTestInvoice(t *testing.T, tx *gorm.DB, organizationID, clientID, invoiceNumber string, amountMinor int64) string {
	t.Helper()
	return insertTestID(t, tx, `INSERT INTO invoices (organization_id, client_id, invoice_number, status, currency_code, amount_minor, issued_on, due_on)
		VALUES (?, ?, ?, 'open', 'USD', ?, '2026-11-01', '2026-11-30') RETURNING id::text AS id`, organizationID, clientID, invoiceNumber, amountMinor)
}

func insertTestID(t *testing.T, tx *gorm.DB, query string, values ...any) string {
	t.Helper()
	row := struct{ ID string }{}
	if err := tx.Raw(query, values...).Scan(&row).Error; err != nil {
		t.Fatalf("insert test fixture: %v", err)
	}
	if row.ID == "" {
		t.Fatal("insert test fixture returned an empty ID")
	}
	return row.ID
}
