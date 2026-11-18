package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/portfolio/ledgerops/backend/internal/config"
	"github.com/portfolio/ledgerops/backend/internal/database"
	"github.com/portfolio/ledgerops/backend/internal/features/reminders"
	ledgersync "github.com/portfolio/ledgerops/backend/internal/features/sync"
	"github.com/portfolio/ledgerops/backend/internal/jobs"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		logger.Error("validate configuration", "error", err)
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect to database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := database.Close(db); err != nil {
			logger.Error("close database", "error", err)
		}
	}()

	worker := jobs.NewWorker(ledgersync.NewPostgresRepository(db), logger, cfg.WorkerPollInterval)
	reminderService := reminders.NewService(reminders.NewPostgresRepository(db))
	reminderScheduler := reminders.NewScheduler(reminderService, logger, cfg.ReminderPollInterval)
	if cfg.Email.Configured() {
		sender, err := reminders.NewSMTPProvider(reminders.SMTPOptions{
			Host: cfg.Email.SMTPHost, Port: cfg.Email.SMTPPort,
			Username: cfg.Email.SMTPUsername, Password: cfg.Email.SMTPPassword,
			FromAddress: cfg.Email.FromAddress, FromName: cfg.Email.FromName,
			RequireTLS: cfg.Email.RequireTLS,
		})
		if err != nil {
			logger.Error("configure SMTP delivery", "error", err)
			os.Exit(1)
		}
		deliveryWorker := reminders.NewDeliveryWorker(reminders.NewPostgresRepository(db), sender, logger, cfg.Email.DeliveryPollInterval)
		go deliveryWorker.Run(ctx)
		logger.Info("email delivery enabled", "from", cfg.Email.FromAddress, "poll_interval", cfg.Email.DeliveryPollInterval)
	} else {
		logger.Info("email delivery disabled; reminders will remain queued")
	}
	logger.Info("worker started", "sync_poll_interval", cfg.WorkerPollInterval, "reminder_poll_interval", cfg.ReminderPollInterval)
	go worker.Run(ctx)
	reminderScheduler.Run(ctx)
	time.Sleep(100 * time.Millisecond)
}
