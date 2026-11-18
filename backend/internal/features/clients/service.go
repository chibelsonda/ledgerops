package clients

import (
	"context"
	"net/mail"
	"strings"
)

type Repository interface {
	List(context.Context, string, ListParams) (Page, error)
	Create(context.Context, string, Actor, MutationInput) (Client, error)
	Update(context.Context, string, string, Actor, MutationInput) (Client, error)
	ListContacts(context.Context, string, string) ([]Contact, error)
	CreateContact(context.Context, string, string, Actor, ContactInput) (Contact, error)
	UpdateContact(context.Context, string, string, Actor, ContactInput) (Contact, error)
}

type ValidationError struct{ Message string }

func (e ValidationError) Error() string { return e.Message }

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }

func (s *Service) List(ctx context.Context, organizationID string, params ListParams) (Page, error) {
	return s.repository.List(ctx, organizationID, params)
}

func (s *Service) Create(ctx context.Context, organizationID string, actor Actor, input MutationInput) (Client, error) {
	normalizeMutation(&input)
	if message := validateMutation(input, true); message != "" {
		return Client{}, ValidationError{message}
	}
	return s.repository.Create(ctx, organizationID, actor, input)
}

func (s *Service) Update(ctx context.Context, organizationID, clientID string, actor Actor, input MutationInput) (Client, error) {
	normalizeMutation(&input)
	if message := validateMutation(input, false); message != "" {
		return Client{}, ValidationError{message}
	}
	return s.repository.Update(ctx, organizationID, clientID, actor, input)
}

func (s *Service) ListContacts(ctx context.Context, organizationID, clientID string) ([]Contact, error) {
	return s.repository.ListContacts(ctx, organizationID, clientID)
}

func (s *Service) CreateContact(ctx context.Context, organizationID, clientID string, actor Actor, input ContactInput) (Contact, error) {
	normalizeContact(&input)
	if message := validateContact(input); message != "" {
		return Contact{}, ValidationError{message}
	}
	return s.repository.CreateContact(ctx, organizationID, clientID, actor, input)
}

func (s *Service) UpdateContact(ctx context.Context, organizationID, contactID string, actor Actor, input ContactInput) (Contact, error) {
	normalizeContact(&input)
	if message := validateContact(input); message != "" {
		return Contact{}, ValidationError{message}
	}
	return s.repository.UpdateContact(ctx, organizationID, contactID, actor, input)
}

func normalizeMutation(input *MutationInput) {
	input.Name = strings.TrimSpace(input.Name)
	input.AuditNote = strings.TrimSpace(input.AuditNote)
	if input.PrimaryContact != nil {
		input.PrimaryContact.Name = strings.TrimSpace(input.PrimaryContact.Name)
		input.PrimaryContact.Email = strings.ToLower(strings.TrimSpace(input.PrimaryContact.Email))
		input.PrimaryContact.Phone = strings.TrimSpace(input.PrimaryContact.Phone)
		input.PrimaryContact.RoleTitle = strings.TrimSpace(input.PrimaryContact.RoleTitle)
	}
}

func validateMutation(input MutationInput, creating bool) string {
	if len(input.Name) < 2 || len(input.Name) > 120 {
		return "Client name must contain 2 to 120 characters."
	}
	if input.PaymentTermsDays < 0 || input.PaymentTermsDays > 365 {
		return "Payment terms must be between 0 and 365 days."
	}
	if input.ReminderDelayDays < 0 || input.ReminderDelayDays > 90 {
		return "Reminder delay must be between 0 and 90 days."
	}
	if !creating && (len(input.AuditNote) < 3 || len(input.AuditNote) > 500) {
		return "Audit note must contain 3 to 500 characters."
	}
	if creating {
		if input.PrimaryContact == nil {
			return "Add a primary billing contact."
		}
		return validateContact(ContactInput{Name: input.PrimaryContact.Name, Email: input.PrimaryContact.Email, Phone: input.PrimaryContact.Phone, RoleTitle: input.PrimaryContact.RoleTitle, IsPrimary: true})
	}
	return ""
}

func normalizeContact(input *ContactInput) {
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.Phone = strings.TrimSpace(input.Phone)
	input.RoleTitle = strings.TrimSpace(input.RoleTitle)
}

func validateContact(input ContactInput) string {
	if len(input.Name) < 2 || len(input.Name) > 100 {
		return "Contact name must contain 2 to 100 characters."
	}
	address, err := mail.ParseAddress(input.Email)
	if err != nil || address.Address != input.Email || len(input.Email) > 254 {
		return "Enter a valid contact email address."
	}
	if len(input.Phone) > 40 || len(input.RoleTitle) > 80 {
		return "Contact phone and role are too long."
	}
	return ""
}
