package financial

import "time"

type AgingBucket string

const (
	Current    AgingBucket = "current"
	Days1To30  AgingBucket = "1-30"
	Days31To60 AgingBucket = "31-60"
	Days61Plus AgingBucket = "61+"
)

func BucketFor(dueDate, asOf time.Time) AgingBucket {
	due := dateOnly(dueDate)
	now := dateOnly(asOf)
	days := int(now.Sub(due).Hours() / 24)
	switch {
	case days <= 0:
		return Current
	case days <= 30:
		return Days1To30
	case days <= 60:
		return Days31To60
	default:
		return Days61Plus
	}
}

func VarianceMinor(budgetMinor, actualMinor int64) int64 {
	return budgetMinor - actualMinor
}

// InvoiceStatus derives the ledger-controlled status from the effective balance.
// Draft and void lifecycle states are intentionally handled by their owning use cases.
func InvoiceStatus(effectiveAmountMinor, paidMinor int64, dueOn, asOf time.Time) string {
	switch {
	case paidMinor == effectiveAmountMinor:
		return "paid"
	case paidMinor > 0:
		return "partial"
	case dateOnly(dueOn).Before(dateOnly(asOf)):
		return "overdue"
	default:
		return "open"
	}
}

// ReconcileInvoiceMutation protects ledger-derived states when commercial fields change.
func ReconcileInvoiceMutation(requestedStatus string, amountMinor, creditMinor, paidMinor int64) (string, bool) {
	effectiveAmount := amountMinor - creditMinor
	if effectiveAmount < paidMinor {
		return "", false
	}
	if effectiveAmount == paidMinor && creditMinor > 0 {
		return "paid", true
	}
	if paidMinor > 0 {
		return "partial", true
	}
	if requestedStatus == "paid" || requestedStatus == "partial" {
		return "", false
	}
	return requestedStatus, true
}

func StatusAfterPayment(effectiveAmountMinor, paidMinor, paymentMinor int64) (string, bool) {
	remaining := effectiveAmountMinor - paidMinor
	if paymentMinor <= 0 || remaining <= 0 || paymentMinor > remaining {
		return "", false
	}
	if paidMinor+paymentMinor == effectiveAmountMinor {
		return "paid", true
	}
	return "partial", true
}

func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
