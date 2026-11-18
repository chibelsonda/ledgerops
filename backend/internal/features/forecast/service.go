package forecast

import (
	"context"
	"strings"
	"time"
)

type Repository interface {
	ListScenarios(context.Context, string) ([]Scenario, error)
	Get(context.Context, string, string) (Forecast, error)
	CreateScenario(context.Context, string, Actor, ScenarioInput) (Forecast, error)
	AddAdjustment(context.Context, string, string, Actor, AdjustmentInput) (Forecast, error)
}

type ValidationError struct{ Message string }

func (e ValidationError) Error() string { return e.Message }

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }

func (s *Service) ListScenarios(ctx context.Context, organizationID string) ([]Scenario, error) {
	return s.repository.ListScenarios(ctx, organizationID)
}
func (s *Service) Get(ctx context.Context, organizationID, scenarioID string) (Forecast, error) {
	return s.repository.Get(ctx, organizationID, scenarioID)
}

func (s *Service) CreateScenario(ctx context.Context, organizationID string, actor Actor, input ScenarioInput) (Forecast, error) {
	input.Name = strings.TrimSpace(input.Name)
	if len(input.Name) < 2 || len(input.Name) > 80 {
		return Forecast{}, ValidationError{Message: "Scenario name must contain 2 to 80 characters."}
	}
	if input.CollectionRateBPS < 0 || input.CollectionRateBPS > 10000 {
		return Forecast{}, ValidationError{Message: "Collection rate must be between 0% and 100%."}
	}
	if input.CollectionDelayDays < 0 || input.CollectionDelayDays > 180 {
		return Forecast{}, ValidationError{Message: "Collection delay must be between 0 and 180 days."}
	}
	if input.ExpenseGrowthBPS < -10000 || input.ExpenseGrowthBPS > 50000 {
		return Forecast{}, ValidationError{Message: "Expense growth must be between -100% and 500%."}
	}
	if input.MilestoneProbabilityBPS < 0 || input.MilestoneProbabilityBPS > 10000 {
		return Forecast{}, ValidationError{Message: "Milestone probability must be between 0% and 100%."}
	}
	return s.repository.CreateScenario(ctx, organizationID, actor, input)
}

func (s *Service) AddAdjustment(ctx context.Context, organizationID, scenarioID string, actor Actor, input AdjustmentInput) (Forecast, error) {
	input.Direction = strings.ToLower(strings.TrimSpace(input.Direction))
	input.Label = strings.TrimSpace(input.Label)
	input.WeekStart = strings.TrimSpace(input.WeekStart)
	if input.Direction != "inflow" && input.Direction != "outflow" {
		return Forecast{}, ValidationError{Message: "Adjustment direction must be inflow or outflow."}
	}
	if len(input.Label) < 2 || len(input.Label) > 120 {
		return Forecast{}, ValidationError{Message: "Adjustment label must contain 2 to 120 characters."}
	}
	if input.AmountMinor <= 0 || input.AmountMinor > 100_000_000_000_000 {
		return Forecast{}, ValidationError{Message: "Adjustment amount must be greater than zero and within the supported range."}
	}
	week, err := time.Parse(time.DateOnly, input.WeekStart)
	if err != nil || week.Weekday() != time.Monday {
		return Forecast{}, ValidationError{Message: "Forecast week must be a Monday."}
	}
	start := monday(time.Now().UTC())
	if week.Before(start) || week.After(start.AddDate(0, 0, 84)) {
		return Forecast{}, ValidationError{Message: "Adjustment week must fall inside the 13-week forecast horizon."}
	}
	return s.repository.AddAdjustment(ctx, organizationID, scenarioID, actor, input)
}

func monday(value time.Time) time.Time {
	date := time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
	offset := (int(date.Weekday()) + 6) % 7
	return date.AddDate(0, 0, -offset)
}
