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
