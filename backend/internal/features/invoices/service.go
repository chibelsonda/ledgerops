package invoices

import (
	"context"
	"fmt"

	"github.com/portfolio/ledgerops/backend/internal/platform/apperrors"
)

// Repository is the persistence boundary required by the invoice use cases.
// Its interface lives beside the consumer so storage implementations remain replaceable.
type Repository interface {
	List(context.Context, string, ListParams) (Page, error)
	ListClients(context.Context, string) ([]ClientOption, error)
	Create(context.Context, string, string, MutationInput) (Invoice, error)
	Update(context.Context, string, string, string, MutationInput) (Invoice, error)
	ListAuditEvents(context.Context, string, string) ([]AuditEvent, error)
	ListPayments(context.Context, string, string) ([]Payment, error)
	RecordPayment(context.Context, string, string, Actor, PaymentInput) (PaymentResult, error)
}

type ValidationError = apperrors.ValidationError

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }

func (s *Service) List(ctx context.Context, organizationID string, params ListParams) (Page, error) {
	return s.repository.List(ctx, organizationID, params)
}

func (s *Service) ListClients(ctx context.Context, organizationID string) ([]ClientOption, error) {
	return s.repository.ListClients(ctx, organizationID)
}

func (s *Service) Create(ctx context.Context, organizationID, actor string, input MutationInput) (Invoice, error) {
	normalizeMutation(&input)
	if message := validateMutation(input, false); message != "" {
		return Invoice{}, ValidationError{Message: message}
	}
	if input.Status == "partial" || input.Status == "paid" {
		return Invoice{}, ValidationError{Message: "Partial and paid statuses are derived from approved ledger entries."}
	}
	return s.repository.Create(ctx, organizationID, actor, input)
}

func (s *Service) Update(ctx context.Context, organizationID, invoiceID, actor string, input MutationInput) (Invoice, error) {
	normalizeMutation(&input)
	if message := validateMutation(input, true); message != "" {
		return Invoice{}, ValidationError{Message: message}
	}
	return s.repository.Update(ctx, organizationID, invoiceID, actor, input)
}

func (s *Service) ListAuditEvents(ctx context.Context, organizationID, invoiceID string) ([]AuditEvent, error) {
	return s.repository.ListAuditEvents(ctx, organizationID, invoiceID)
}

func (s *Service) ListPayments(ctx context.Context, organizationID, invoiceID string) ([]Payment, error) {
	return s.repository.ListPayments(ctx, organizationID, invoiceID)
}

func (s *Service) RecordPayment(ctx context.Context, organizationID, invoiceID string, actor Actor, input PaymentInput) (PaymentResult, error) {
	normalizePayment(&input)
	if message := validatePayment(input); message != "" {
		return PaymentResult{}, ValidationError{Message: message}
	}
	result, err := s.repository.RecordPayment(ctx, organizationID, invoiceID, actor, input)
	if err != nil {
		return PaymentResult{}, fmt.Errorf("record payment: %w", err)
	}
	return result, nil
}
