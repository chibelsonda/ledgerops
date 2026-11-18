package reminders

import (
	"context"
	"log/slog"
	"time"
)

const deliveryLease = 10 * time.Minute

type DeliveryQueue interface {
	ClaimDelivery(context.Context, time.Duration) (*Delivery, error)
	MarkDelivered(context.Context, string, string) error
	MarkDeliveryFailed(context.Context, Delivery, error) error
}

type EmailSender interface {
	Send(context.Context, EmailMessage) (string, error)
}

type DeliveryWorker struct {
	queue        DeliveryQueue
	sender       EmailSender
	logger       *slog.Logger
	pollInterval time.Duration
}

func NewDeliveryWorker(queue DeliveryQueue, sender EmailSender, logger *slog.Logger, pollInterval time.Duration) *DeliveryWorker {
	return &DeliveryWorker{queue: queue, sender: sender, logger: logger, pollInterval: pollInterval}
}

func (w *DeliveryWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	w.processOne(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processOne(ctx)
		}
	}
}

func (w *DeliveryWorker) processOne(ctx context.Context) {
	delivery, err := w.queue.ClaimDelivery(ctx, deliveryLease)
	if err != nil {
		w.logger.Error("claim reminder delivery", "error", err)
		return
	}
	if delivery == nil {
		return
	}
	messageID, err := w.sender.Send(ctx, EmailMessage{
		To: delivery.RecipientEmail, Subject: delivery.Subject, Body: delivery.Message,
		MessageID: "<" + delivery.ID + "@ledgerops.local>",
	})
	if err != nil {
		w.logger.Warn("send reminder", "reminder_id", delivery.ID, "attempt", delivery.Attempts, "error", err)
		if failErr := w.queue.MarkDeliveryFailed(ctx, *delivery, err); failErr != nil {
			w.logger.Error("reschedule reminder", "reminder_id", delivery.ID, "error", failErr)
		}
		return
	}
	if err := w.queue.MarkDelivered(ctx, delivery.ID, messageID); err != nil {
		w.logger.Error("complete reminder delivery", "reminder_id", delivery.ID, "error", err)
		return
	}
	w.logger.Info("reminder sent", "reminder_id", delivery.ID, "attempt", delivery.Attempts)
}
