package close

import (
	"context"
	"strings"
	"time"
)

type Repository interface {
	List(context.Context, string) ([]PeriodSummary, error)
	Get(context.Context, string, string) (Detail, error)
	Create(context.Context, string, Actor, CreateInput) (Detail, error)
	UpdateTask(context.Context, string, string, Actor, TaskInput) (Detail, error)
	Submit(context.Context, string, string, Actor) (Detail, error)
	Decide(context.Context, string, string, Actor, DecisionInput) (Detail, error)
}

type ValidationError struct{ Message string }

func (e ValidationError) Error() string { return e.Message }

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }

func (s *Service) List(ctx context.Context, organizationID string) ([]PeriodSummary, error) {
	return s.repository.List(ctx, organizationID)
}

func (s *Service) Get(ctx context.Context, organizationID, periodID string) (Detail, error) {
	return s.repository.Get(ctx, organizationID, periodID)
}

func (s *Service) Create(ctx context.Context, organizationID string, actor Actor, input CreateInput) (Detail, error) {
	input.MonthStart = strings.TrimSpace(input.MonthStart)
	input.DueOn = strings.TrimSpace(input.DueOn)
	month, err := time.Parse(time.DateOnly, input.MonthStart)
	if err != nil || month.Day() != 1 {
		return Detail{}, ValidationError{Message: "Close month must be the first day of a month."}
	}
	due, err := time.Parse(time.DateOnly, input.DueOn)
	if err != nil || due.Before(month) {
		return Detail{}, ValidationError{Message: "Due date must be on or after the close month."}
	}
	return s.repository.Create(ctx, organizationID, actor, input)
}

func (s *Service) UpdateTask(ctx context.Context, organizationID, taskID string, actor Actor, input TaskInput) (Detail, error) {
	input.Status = strings.ToLower(strings.TrimSpace(input.Status))
	input.EvidenceNote = strings.TrimSpace(input.EvidenceNote)
	if input.Status != "pending" && input.Status != "completed" {
		return Detail{}, ValidationError{Message: "Task status must be pending or completed."}
	}
	if input.Status == "completed" && (len(input.EvidenceNote) < 3 || len(input.EvidenceNote) > 500) {
		return Detail{}, ValidationError{Message: "Completion evidence must contain 3 to 500 characters."}
	}
	return s.repository.UpdateTask(ctx, organizationID, taskID, actor, input)
}

func (s *Service) Submit(ctx context.Context, organizationID, periodID string, actor Actor) (Detail, error) {
	return s.repository.Submit(ctx, organizationID, periodID, actor)
}

func (s *Service) Decide(ctx context.Context, organizationID, periodID string, actor Actor, input DecisionInput) (Detail, error) {
	input.Decision = strings.ToLower(strings.TrimSpace(input.Decision))
	input.Note = strings.TrimSpace(input.Note)
	if input.Decision != "approved" && input.Decision != "rejected" {
		return Detail{}, ValidationError{Message: "Decision must be approved or rejected."}
	}
	if len(input.Note) < 3 || len(input.Note) > 500 {
		return Detail{}, ValidationError{Message: "Review note must contain 3 to 500 characters."}
	}
	return s.repository.Decide(ctx, organizationID, periodID, actor, input)
}
