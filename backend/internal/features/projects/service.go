package projects

import (
	"context"

	"github.com/portfolio/ledgerops/backend/internal/platform/apperrors"
)

type Repository interface {
	List(context.Context, string) ([]ProjectSummary, error)
	ListClients(context.Context, string) ([]ClientOption, error)
	Get(context.Context, string, string) (Detail, error)
	Create(context.Context, string, Actor, CreateInput) (Detail, error)
	DecideContract(context.Context, string, string, Actor, DecisionInput) (Detail, error)
	CreateChangeOrder(context.Context, string, string, Actor, ChangeOrderInput) (Detail, error)
	DecideChangeOrder(context.Context, string, string, Actor, DecisionInput) (Detail, error)
	CreateMilestone(context.Context, string, string, Actor, MilestoneInput) (Detail, error)
	InvoiceMilestone(context.Context, string, string, Actor, InvoiceInput) (Detail, error)
}

type ValidationError = apperrors.ValidationError

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }

func (s *Service) List(ctx context.Context, organizationID string) ([]ProjectSummary, error) {
	return s.repository.List(ctx, organizationID)
}

func (s *Service) ListClients(ctx context.Context, organizationID string) ([]ClientOption, error) {
	return s.repository.ListClients(ctx, organizationID)
}

func (s *Service) Get(ctx context.Context, organizationID, projectID string) (Detail, error) {
	return s.repository.Get(ctx, organizationID, projectID)
}

func (s *Service) Create(ctx context.Context, organizationID string, actor Actor, input CreateInput) (Detail, error) {
	normalizeCreate(&input)
	if message := validateCreate(input); message != "" {
		return Detail{}, ValidationError{Message: message}
	}
	return s.repository.Create(ctx, organizationID, actor, input)
}

func (s *Service) DecideContract(ctx context.Context, organizationID, contractID string, actor Actor, input DecisionInput) (Detail, error) {
	normalizeDecision(&input)
	if message := validateDecision(input); message != "" {
		return Detail{}, ValidationError{Message: message}
	}
	return s.repository.DecideContract(ctx, organizationID, contractID, actor, input)
}

func (s *Service) CreateChangeOrder(ctx context.Context, organizationID, contractID string, actor Actor, input ChangeOrderInput) (Detail, error) {
	normalizeChangeOrder(&input)
	if message := validateChangeOrder(input); message != "" {
		return Detail{}, ValidationError{Message: message}
	}
	return s.repository.CreateChangeOrder(ctx, organizationID, contractID, actor, input)
}

func (s *Service) DecideChangeOrder(ctx context.Context, organizationID, changeOrderID string, actor Actor, input DecisionInput) (Detail, error) {
	normalizeDecision(&input)
	if message := validateDecision(input); message != "" {
		return Detail{}, ValidationError{Message: message}
	}
	return s.repository.DecideChangeOrder(ctx, organizationID, changeOrderID, actor, input)
}

func (s *Service) CreateMilestone(ctx context.Context, organizationID, contractID string, actor Actor, input MilestoneInput) (Detail, error) {
	normalizeMilestone(&input)
	if message := validateMilestone(input); message != "" {
		return Detail{}, ValidationError{Message: message}
	}
	return s.repository.CreateMilestone(ctx, organizationID, contractID, actor, input)
}

func (s *Service) InvoiceMilestone(ctx context.Context, organizationID, milestoneID string, actor Actor, input InvoiceInput) (Detail, error) {
	normalizeInvoice(&input)
	if message := validateInvoice(input); message != "" {
		return Detail{}, ValidationError{Message: message}
	}
	return s.repository.InvoiceMilestone(ctx, organizationID, milestoneID, actor, input)
}
