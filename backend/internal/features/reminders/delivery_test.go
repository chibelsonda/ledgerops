package reminders

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"
)

type deliveryQueueStub struct {
	delivery    *Delivery
	deliveredID string
	messageID   string
	failedID    string
}

func (s *deliveryQueueStub) ClaimDelivery(context.Context, time.Duration) (*Delivery, error) {
	return s.delivery, nil
}
func (s *deliveryQueueStub) MarkDelivered(_ context.Context, id, messageID string) error {
	s.deliveredID, s.messageID = id, messageID
	return nil
}
func (s *deliveryQueueStub) MarkDeliveryFailed(_ context.Context, delivery Delivery, _ error) error {
	s.failedID = delivery.ID
	return nil
}

type emailSenderStub struct {
	err     error
	message EmailMessage
}

func (s *emailSenderStub) Send(_ context.Context, message EmailMessage) (string, error) {
	s.message = message
	return message.MessageID, s.err
}

func TestDeliveryWorkerMarksAcceptedMessageSent(t *testing.T) {
	queue := &deliveryQueueStub{delivery: &Delivery{ID: "reminder-1", RecipientEmail: "client@example.com", Attempts: 1, MaxAttempts: 5}}
	sender := &emailSenderStub{}
	worker := NewDeliveryWorker(queue, sender, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Second)
	worker.processOne(context.Background())
	if queue.deliveredID != "reminder-1" || queue.messageID != "<reminder-1@ledgerops.local>" || queue.failedID != "" {
		t.Fatalf("unexpected queue result: %#v", queue)
	}
	if sender.message.MessageID != "<reminder-1@ledgerops.local>" {
		t.Fatalf("MessageID = %q, want stable reminder ID", sender.message.MessageID)
	}
}

func TestDeliveryWorkerReschedulesProviderFailure(t *testing.T) {
	queue := &deliveryQueueStub{delivery: &Delivery{ID: "reminder-2", RecipientEmail: "client@example.com", Attempts: 2, MaxAttempts: 5}}
	worker := NewDeliveryWorker(queue, &emailSenderStub{err: errors.New("SMTP unavailable")}, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Second)
	worker.processOne(context.Background())
	if queue.failedID != "reminder-2" || queue.deliveredID != "" {
		t.Fatalf("unexpected queue result: %#v", queue)
	}
}

func TestWebhookSecretComparison(t *testing.T) {
	if !sameSecret("shared-secret", "shared-secret") || sameSecret("wrong", "shared-secret") {
		t.Fatal("constant-time secret comparison returned an unexpected result")
	}
}
