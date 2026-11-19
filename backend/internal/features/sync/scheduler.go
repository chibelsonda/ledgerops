package sync

import (
	"context"
	"log/slog"
	"time"
)

type ScheduleRepository interface {
	EnqueueDue(context.Context, int) (int, error)
}

type Scheduler struct {
	repository ScheduleRepository
	logger     *slog.Logger
	interval   time.Duration
}

func NewScheduler(repository ScheduleRepository, logger *slog.Logger, interval time.Duration) *Scheduler {
	return &Scheduler{repository: repository, logger: logger, interval: interval}
}

func (s *Scheduler) Run(ctx context.Context) {
	s.runOnce(ctx)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runOnce(ctx)
		}
	}
}

func (s *Scheduler) runOnce(ctx context.Context) {
	count, err := s.repository.EnqueueDue(ctx, 25)
	if err != nil {
		s.logger.Error("enqueue scheduled QuickBooks synchronization", "error", err)
		return
	}
	if count > 0 {
		s.logger.Info("scheduled QuickBooks synchronization", "jobs", count)
	}
}
