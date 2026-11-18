package reminders

import (
	"context"
	"log/slog"
	"time"
)

type Generator interface {
	GenerateAllDue(context.Context) (int64, error)
}
type Scheduler struct {
	generator Generator
	logger    *slog.Logger
	interval  time.Duration
}

func NewScheduler(generator Generator, logger *slog.Logger, interval time.Duration) *Scheduler {
	return &Scheduler{generator: generator, logger: logger, interval: interval}
}
func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	s.generate(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.generate(ctx)
		}
	}
}
func (s *Scheduler) generate(ctx context.Context) {
	count, err := s.generator.GenerateAllDue(ctx)
	if err != nil {
		s.logger.Error("generate overdue reminders", "error", err)
		return
	}
	if count > 0 {
		s.logger.Info("overdue reminders queued", "count", count)
	}
}
