package forecast

import (
	"context"
	"errors"
	"testing"
	"time"
)

type repositoryStub struct{ created, adjusted bool }

func (r *repositoryStub) ListScenarios(context.Context, string) ([]Scenario, error) { return nil, nil }
func (r *repositoryStub) Get(context.Context, string, string) (Forecast, error) {
	return Forecast{}, nil
}
func (r *repositoryStub) CreateScenario(context.Context, string, Actor, ScenarioInput) (Forecast, error) {
	r.created = true
	return Forecast{}, nil
}
func (r *repositoryStub) AddAdjustment(context.Context, string, string, Actor, AdjustmentInput) (Forecast, error) {
	r.adjusted = true
	return Forecast{}, nil
}

func TestScenarioRejectsInvalidRate(t *testing.T) {
	repository := &repositoryStub{}
	_, err := NewService(repository).CreateScenario(context.Background(), "org", Actor{}, ScenarioInput{Name: "Invalid", CollectionRateBPS: 10001})
	var validation ValidationError
	if !errors.As(err, &validation) || repository.created {
		t.Fatalf("expected rate validation, got %v", err)
	}
}

func TestAdjustmentRequiresForecastMonday(t *testing.T) {
	repository := &repositoryStub{}
	_, err := NewService(repository).AddAdjustment(context.Background(), "org", "scenario", Actor{}, AdjustmentInput{WeekStart: "2026-11-18", Direction: "outflow", Label: "Insurance", AmountMinor: 100})
	var validation ValidationError
	if !errors.As(err, &validation) || repository.adjusted {
		t.Fatalf("expected Monday validation, got %v", err)
	}
}

func TestValidAdjustmentReachesRepository(t *testing.T) {
	repository := &repositoryStub{}
	week := monday(time.Now().UTC()).Format(time.DateOnly)
	_, err := NewService(repository).AddAdjustment(context.Background(), "org", "scenario", Actor{}, AdjustmentInput{WeekStart: week, Direction: " Inflow ", Label: " Owner funding ", AmountMinor: 100})
	if err != nil || !repository.adjusted {
		t.Fatalf("expected normalized adjustment to reach repository, got %v", err)
	}
}
