package adjustments

import "strings"

func normalizeCreate(input *CreateInput) {
	input.Kind = Kind(strings.ToLower(strings.TrimSpace(string(input.Kind))))
	input.PaymentID = strings.TrimSpace(input.PaymentID)
	input.Reason = strings.TrimSpace(input.Reason)
}

func validateCreate(input CreateInput) string {
	if len(input.Reason) < 3 || len(input.Reason) > 500 {
		return "Justification must contain 3 to 500 characters."
	}
	switch input.Kind {
	case KindCreditNote:
		if input.AmountMinor <= 0 || input.AmountMinor > 100_000_000_000_000 {
			return "Credit amount must be greater than zero and within the supported range."
		}
		if input.PaymentID != "" {
			return "Credit notes cannot reference a payment."
		}
	case KindPaymentReversal:
		if !validUUID(input.PaymentID) {
			return "Select a valid payment to reverse."
		}
		if input.AmountMinor != 0 {
			return "Payment reversal amounts are derived from the selected payment."
		}
	default:
		return "Select a valid adjustment type."
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
	if len(input.Note) < 3 || len(input.Note) > 500 {
		return "Review note must contain 3 to 500 characters."
	}
	return ""
}

func validUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for index, char := range value {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if char != '-' {
				return false
			}
			continue
		}
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}
