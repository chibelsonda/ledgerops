package close

import (
	"context"
	"errors"
	"testing"
)

type repositoryStub struct{ created, updated, decided bool }

func (r *repositoryStub) List(context.Context, string) ([]PeriodSummary, error) { return nil, nil }
func (r *repositoryStub) Get(context.Context, string, string) (Detail, error)   { return Detail{}, nil }
func (r *repositoryStub) Create(context.Context, string, Actor, CreateInput) (Detail, error) {
	r.created = true
	return Detail{}, nil
}
func (r *repositoryStub) UpdateTask(context.Context, string, string, Actor, TaskInput) (Detail, error) {
	r.updated = true
	return Detail{}, nil
}
func (r *repositoryStub) Submit(context.Context, string, string, Actor) (Detail, error) {
	return Detail{}, nil
}
func (r *repositoryStub) Decide(context.Context, string, string, Actor, DecisionInput) (Detail, error) {
	r.decided = true
	return Detail{}, nil
}

func TestCreateRequiresMonthStart(t *testing.T) {
	repository := &repositoryStub{}
	_, err := NewService(repository).Create(context.Background(), "org", Actor{}, CreateInput{MonthStart: "2026-11-02", DueOn: "2026-12-05"})
	var validation ValidationError
	if !errors.As(err, &validation) || repository.created {
		t.Fatalf("expected validation before repository, got %v", err)
	}
}

func TestCompletedTaskRequiresEvidence(t *testing.T) {
	repository := &repositoryStub{}
	_, err := NewService(repository).UpdateTask(context.Background(), "org", "task", Actor{}, TaskInput{Status: "completed", EvidenceNote: "no"})
	var validation ValidationError
	if !errors.As(err, &validation) || repository.updated {
		t.Fatalf("expected evidence validation, got %v", err)
	}
}

func TestValidDecisionReachesRepository(t *testing.T) {
	repository := &repositoryStub{}
	_, err := NewService(repository).Decide(context.Background(), "org", "period", Actor{}, DecisionInput{Decision: " APPROVED ", Note: " Close binder verified. "})
	if err != nil || !repository.decided {
		t.Fatalf("expected normalized decision to reach repository, got %v", err)
	}
}
