package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/portfolio/ledgerops/backend/internal/features/reminders"
)

type passwordResetMailer struct {
	sender *reminders.SMTPProvider
}

func (m passwordResetMailer) SendPasswordReset(ctx context.Context, email, displayName, resetURL string) error {
	name := strings.TrimSpace(displayName)
	if name == "" {
		name = "there"
	}
	body := fmt.Sprintf("Hello %s,\n\nUse the link below to reset your LedgerOps password. This link expires in 30 minutes and can only be used once.\n\n%s\n\nIf you did not request this change, you can ignore this email.", name, resetURL)
	_, err := m.sender.Send(ctx, reminders.EmailMessage{
		To: email, Subject: "Reset your LedgerOps password", Body: body,
	})
	return err
}
