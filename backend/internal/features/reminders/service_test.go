package reminders

import (
	"context"
	"errors"
	"testing"
	"time"
)

type serviceRepositoryStub struct {
	recorded DeliveryEventInput
}

func (s *serviceRepositoryStub) List(context.Context, string) ([]Reminder, error)    { return nil, nil }
func (s *serviceRepositoryStub) GenerateDue(context.Context, string) (int64, error)  { return 0, nil }
func (s *serviceRepositoryStub) GenerateAllDue(context.Context) (int64, error)       { return 0, nil }
func (s *serviceRepositoryStub) Cancel(context.Context, string, string, Actor) error { return nil }
func (s *serviceRepositoryStub) RecordDeliveryEvent(_ context.Context, input DeliveryEventInput, _ time.Time) error {
	s.recorded = input
	return nil
}

func TestRecordDeliveryEventNormalizesAndPersists(t *testing.T) {
	repository := &serviceRepositoryStub{}
	now := time.Date(2026, time.November, 19, 12, 0, 0, 0, time.UTC)
	service := newServiceWithClock(repository, func() time.Time { return now })
	input := DeliveryEventInput{
		EventID: " provider-event-1 ", ProviderMessageID: " <message@ledgerops.local> ",
		EventType: " DELIVERED ", OccurredAt: now.Add(-time.Minute).Format(time.RFC3339),
	}
	if err := service.RecordDeliveryEvent(context.Background(), input); err != nil {
		t.Fatalf("RecordDeliveryEvent() error = %v", err)
	}
	if repository.recorded.EventID != "provider-event-1" || repository.recorded.EventType != "delivered" {
		t.Fatalf("recorded event was not normalized: %#v", repository.recorded)
	}
}

func TestRecordDeliveryEventRejectsInvalidInput(t *testing.T) {
	service := NewService(&serviceRepositoryStub{})
	tests := []DeliveryEventInput{
		{EventID: "", ProviderMessageID: "message", EventType: "delivered", OccurredAt: time.Now().Format(time.RFC3339)},
		{EventID: "event", ProviderMessageID: "message", EventType: "opened", OccurredAt: time.Now().Format(time.RFC3339)},
		{EventID: "event", ProviderMessageID: "message", EventType: "bounced", OccurredAt: "not-a-time"},
	}
	for _, input := range tests {
		if err := service.RecordDeliveryEvent(context.Background(), input); !errors.Is(err, ErrInvalidDeliveryEvent) {
			t.Fatalf("RecordDeliveryEvent(%#v) error = %v, want ErrInvalidDeliveryEvent", input, err)
		}
	}
}
