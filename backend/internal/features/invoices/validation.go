package invoices

import (
	"regexp"
	"strings"
	"time"
)

var (
	currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)
	uuidPattern     = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-57][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
)

func normalizeMutation(input *MutationInput) {
	input.ClientID = strings.TrimSpace(input.ClientID)
	input.InvoiceNumber = strings.TrimSpace(input.InvoiceNumber)
	input.Status = strings.ToLower(strings.TrimSpace(input.Status))
	input.CurrencyCode = strings.ToUpper(strings.TrimSpace(input.CurrencyCode))
	input.IssuedOn = strings.TrimSpace(input.IssuedOn)
	input.DueOn = strings.TrimSpace(input.DueOn)
	input.AuditNote = strings.TrimSpace(input.AuditNote)
}

func validateMutation(input MutationInput, requireAuditNote bool) string {
	if !isUUID(input.ClientID) {
		return "Select a valid client."
	}
	if len(input.InvoiceNumber) < 2 || len(input.InvoiceNumber) > 40 {
		return "Invoice number must contain 2 to 40 characters."
	}
	if !validStatus(input.Status) {
		return "Select a valid invoice status."
	}
	if !currencyPattern.MatchString(input.CurrencyCode) {
		return "Currency must be a three-letter ISO code."
	}
	if input.AmountMinor <= 0 || input.AmountMinor > 100_000_000_000_000 {
		return "Amount must be greater than zero and within the supported range."
	}
	issuedOn, err := time.Parse(time.DateOnly, input.IssuedOn)
	if err != nil {
		return "Issue date must use YYYY-MM-DD format."
	}
	dueOn, err := time.Parse(time.DateOnly, input.DueOn)
	if err != nil {
		return "Due date must use YYYY-MM-DD format."
	}
	if dueOn.Before(issuedOn) {
		return "Due date cannot be before the issue date."
	}
	if requireAuditNote && (len(input.AuditNote) < 3 || len(input.AuditNote) > 500) {
		return "Audit note must contain 3 to 500 characters."
	}
	return ""
}

func validatePayment(input PaymentInput) string {
	if input.AmountMinor <= 0 || input.AmountMinor > 100_000_000_000_000 {
		return "Payment amount must be greater than zero and within the supported range."
	}
	if _, err := time.Parse(time.DateOnly, input.ReceivedOn); err != nil {
		return "Payment date must use YYYY-MM-DD format."
	}
	if len(input.Reference) > 100 {
		return "Payment reference must contain 100 characters or fewer."
	}
	if len(input.AuditNote) < 3 || len(input.AuditNote) > 500 {
		return "Audit note must contain 3 to 500 characters."
	}
	return ""
}

func normalizePayment(input *PaymentInput) {
	input.ReceivedOn = strings.TrimSpace(input.ReceivedOn)
	input.Reference = strings.TrimSpace(input.Reference)
	input.AuditNote = strings.TrimSpace(input.AuditNote)
}

func validStatus(status string) bool {
	switch status {
	case "draft", "open", "partial", "paid", "void", "overdue":
		return true
	default:
		return false
	}
}

func isUUID(value string) bool {
	return len(value) == 36 && uuidPattern.MatchString(value)
}
