package reminders

import (
	"context"
	"errors"
	"strings"
	"time"
)

var ErrInvalidDeliveryEvent = errors.New("invalid delivery event")

type Repository interface {
	List(context.Context, string) ([]Reminder, error)
	GenerateDue(context.Context, string) (int64, error)
	GenerateAllDue(context.Context) (int64, error)
	Cancel(context.Context, string, string, Actor) error
	RecordDeliveryEvent(context.Context, DeliveryEventInput, time.Time) error
}

type Service struct {
	repository Repository
	now        func() time.Time
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository, now: time.Now}
}
func newServiceWithClock(repository Repository, now func() time.Time) *Service {
	return &Service{repository: repository, now: now}
}
func (s *Service) List(ctx context.Context, organizationID string) ([]Reminder, error) {
	return s.repository.List(ctx, organizationID)
}
func (s *Service) GenerateDue(ctx context.Context, organizationID string) (GenerateResult, error) {
	count, err := s.repository.GenerateDue(ctx, organizationID)
	return GenerateResult{Created: count}, err
}
func (s *Service) GenerateAllDue(ctx context.Context) (int64, error) {
	return s.repository.GenerateAllDue(ctx)
}
func (s *Service) Cancel(ctx context.Context, organizationID, reminderID string, actor Actor) error {
	return s.repository.Cancel(ctx, organizationID, reminderID, actor)
}

func (s *Service) RecordDeliveryEvent(ctx context.Context, input DeliveryEventInput) error {
	input.EventID = strings.TrimSpace(input.EventID)
	input.ProviderMessageID = strings.TrimSpace(input.ProviderMessageID)
	input.EventType = strings.ToLower(strings.TrimSpace(input.EventType))
	if input.EventID == "" || len(input.EventID) > 200 || input.ProviderMessageID == "" || len(input.ProviderMessageID) > 998 {
		return ErrInvalidDeliveryEvent
	}
	switch input.EventType {
	case "delivered", "bounced", "complained", "deferred":
	default:
		return ErrInvalidDeliveryEvent
	}
	occurredAt, err := time.Parse(time.RFC3339, input.OccurredAt)
	if err != nil || occurredAt.After(s.now().Add(5*time.Minute)) {
		return ErrInvalidDeliveryEvent
	}
	return s.repository.RecordDeliveryEvent(ctx, input, occurredAt)
}
