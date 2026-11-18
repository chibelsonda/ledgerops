package projects

import (
	"regexp"
	"strings"
	"time"
)

const maxMoneyMinor int64 = 100_000_000_000_000

var (
	projectCurrencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)
	projectUUIDPattern     = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-57][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
)

func normalizeCreate(input *CreateInput) {
	input.ClientID = strings.TrimSpace(input.ClientID)
	input.ProjectCode = strings.ToUpper(strings.TrimSpace(input.ProjectCode))
	input.Name = strings.TrimSpace(input.Name)
	input.CurrencyCode = strings.ToUpper(strings.TrimSpace(input.CurrencyCode))
	input.StartOn = strings.TrimSpace(input.StartOn)
	input.EndOn = strings.TrimSpace(input.EndOn)
	input.ContractNumber = strings.ToUpper(strings.TrimSpace(input.ContractNumber))
	input.ContractTitle = strings.TrimSpace(input.ContractTitle)
}

func validateCreate(input CreateInput) string {
	if !projectUUIDPattern.MatchString(input.ClientID) {
		return "Select a valid client."
	}
	if !lengthBetween(input.ProjectCode, 2, 30) {
		return "Project code must contain 2 to 30 characters."
	}
	if !lengthBetween(input.Name, 2, 150) {
		return "Project name must contain 2 to 150 characters."
	}
	if !projectCurrencyPattern.MatchString(input.CurrencyCode) {
		return "Currency must be a three-letter ISO code."
	}
	start, err := time.Parse(time.DateOnly, input.StartOn)
	if err != nil {
		return "Start date must use YYYY-MM-DD format."
	}
	if input.EndOn != "" {
		end, err := time.Parse(time.DateOnly, input.EndOn)
		if err != nil || end.Before(start) {
			return "End date must be on or after the start date."
		}
	}
	if !lengthBetween(input.ContractNumber, 2, 40) {
		return "Contract number must contain 2 to 40 characters."
	}
	if !lengthBetween(input.ContractTitle, 2, 150) {
		return "Contract title must contain 2 to 150 characters."
	}
	if input.OriginalValueMinor <= 0 || input.OriginalValueMinor > maxMoneyMinor {
		return "Original contract value must be greater than zero and within the supported range."
	}
	return ""
}

func normalizeDecision(input *DecisionInput) {
	input.Decision = strings.ToLower(strings.TrimSpace(input.Decision))
	input.Note = strings.TrimSpace(input.Note)
}

func validateDecision(input DecisionInput) string {
	if input.Decision != "approved" && input.Decision != "rejected" {
		return "Decision must be approved or rejected."
	}
	if !lengthBetween(input.Note, 3, 500) {
		return "Review note must contain 3 to 500 characters."
	}
	return ""
}

func normalizeChangeOrder(input *ChangeOrderInput) {
	input.ChangeOrderNumber = strings.ToUpper(strings.TrimSpace(input.ChangeOrderNumber))
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
}

func validateChangeOrder(input ChangeOrderInput) string {
	if !lengthBetween(input.ChangeOrderNumber, 2, 40) {
		return "Change-order number must contain 2 to 40 characters."
	}
	if !lengthBetween(input.Title, 2, 150) {
		return "Change-order title must contain 2 to 150 characters."
	}
	if !lengthBetween(input.Description, 3, 1000) {
		return "Description must contain 3 to 1000 characters."
	}
	if input.AmountMinor == 0 || input.AmountMinor > maxMoneyMinor || input.AmountMinor < -maxMoneyMinor {
		return "Change amount must be non-zero and within the supported range."
	}
	return ""
}

func normalizeMilestone(input *MilestoneInput) {
	input.Name = strings.TrimSpace(input.Name)
	input.DueOn = strings.TrimSpace(input.DueOn)
}

func validateMilestone(input MilestoneInput) string {
	if !lengthBetween(input.Name, 2, 150) {
		return "Milestone name must contain 2 to 150 characters."
	}
	if input.AmountMinor <= 0 || input.AmountMinor > maxMoneyMinor {
		return "Milestone amount must be greater than zero and within the supported range."
	}
	if _, err := time.Parse(time.DateOnly, input.DueOn); err != nil {
		return "Milestone due date must use YYYY-MM-DD format."
	}
	return ""
}

func normalizeInvoice(input *InvoiceInput) {
	input.InvoiceNumber = strings.ToUpper(strings.TrimSpace(input.InvoiceNumber))
	input.IssuedOn = strings.TrimSpace(input.IssuedOn)
	input.DueOn = strings.TrimSpace(input.DueOn)
}

func validateInvoice(input InvoiceInput) string {
	if !lengthBetween(input.InvoiceNumber, 2, 40) {
		return "Invoice number must contain 2 to 40 characters."
	}
	issued, err := time.Parse(time.DateOnly, input.IssuedOn)
	if err != nil {
		return "Issue date must use YYYY-MM-DD format."
	}
	due, err := time.Parse(time.DateOnly, input.DueOn)
	if err != nil || due.Before(issued) {
		return "Due date must be on or after the issue date."
	}
	return ""
}

func lengthBetween(value string, minimum, maximum int) bool {
	return len(value) >= minimum && len(value) <= maximum
}
