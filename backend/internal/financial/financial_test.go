package financial

import (
	"testing"
	"time"
)

func TestBucketFor(t *testing.T) {
	asOf := time.Date(2026, time.November, 18, 15, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		due  time.Time
		want AgingBucket
	}{
		{"not due", asOf.AddDate(0, 0, 5), Current},
		{"due today", asOf, Current},
		{"fifteen days", asOf.AddDate(0, 0, -15), Days1To30},
		{"forty five days", asOf.AddDate(0, 0, -45), Days31To60},
		{"ninety days", asOf.AddDate(0, 0, -90), Days61Plus},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := BucketFor(test.due, asOf); got != test.want {
				t.Fatalf("BucketFor() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestVarianceMinor(t *testing.T) {
	if got := VarianceMinor(250_000, 212_000); got != 38_000 {
		t.Fatalf("VarianceMinor() = %d, want 38000", got)
	}
}

func TestInvoiceStatus(t *testing.T) {
	asOf := time.Date(2026, time.November, 19, 15, 0, 0, 0, time.UTC)
	tests := []struct {
		name            string
		effective, paid int64
		due             time.Time
		want            string
	}{
		{name: "paid", effective: 10_000, paid: 10_000, due: asOf.AddDate(0, 0, 5), want: "paid"},
		{name: "partial", effective: 10_000, paid: 5_000, due: asOf.AddDate(0, 0, -5), want: "partial"},
		{name: "overdue", effective: 10_000, paid: 0, due: asOf.AddDate(0, 0, -1), want: "overdue"},
		{name: "open", effective: 10_000, paid: 0, due: asOf, want: "open"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := InvoiceStatus(test.effective, test.paid, test.due, asOf); got != test.want {
				t.Fatalf("InvoiceStatus() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestLedgerMutationPolicies(t *testing.T) {
	if _, ok := ReconcileInvoiceMutation("open", 10_000, 1_000, 9_500); ok {
		t.Fatal("invoice amount below the net paid balance was accepted")
	}
	if status, ok := ReconcileInvoiceMutation("open", 10_000, 1_000, 9_000); !ok || status != "paid" {
		t.Fatalf("status = %q, ok = %v; want paid, true", status, ok)
	}
	if _, ok := StatusAfterPayment(10_000, 8_000, 2_001); ok {
		t.Fatal("overpayment was accepted")
	}
	if status, ok := StatusAfterPayment(10_000, 8_000, 2_000); !ok || status != "paid" {
		t.Fatalf("status = %q, ok = %v; want paid, true", status, ok)
	}
}
