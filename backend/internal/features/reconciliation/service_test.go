package reconciliation

import (
	"context"
	"testing"
)

type capturingRepository struct {
	actor string
	note  string
	calls int
}

func (r *capturingRepository) List(context.Context, string) ([]Item, error) { return nil, nil }
func (r *capturingRepository) Resolve(_ context.Context, _, _, actor, note string) error {
	r.actor, r.note = actor, note
	r.calls++
	return nil
}

func TestServiceNormalizesResolutionAndPreservesActor(t *testing.T) {
	repository := &capturingRepository{}
	service := NewService(repository)

	if err := service.Resolve(context.Background(), "organization", "item", "reviewer@ledgerops.demo", "  Confirmed against statement.  "); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if repository.actor != "reviewer@ledgerops.demo" || repository.note != "Confirmed against statement." {
		t.Fatalf("actor = %q, note = %q", repository.actor, repository.note)
	}
}

func TestServiceRejectsInvalidResolutionBeforePersistence(t *testing.T) {
	repository := &capturingRepository{}
	service := NewService(repository)

	if err := service.Resolve(context.Background(), "organization", "item", "reviewer@ledgerops.demo", " x "); err != ErrInvalidResolution {
		t.Fatalf("error = %v, want ErrInvalidResolution", err)
	}
	if repository.calls != 0 {
		t.Fatal("repository called for invalid resolution")
	}
}
