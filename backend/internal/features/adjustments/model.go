package adjustments

type Kind string

const (
	KindCreditNote      Kind = "credit_note"
	KindPaymentReversal Kind = "payment_reversal"
)

type Actor struct {
	UserID string
	Email  string
}

type CreateInput struct {
	Kind        Kind   `json:"kind"`
	AmountMinor int64  `json:"amountMinor,omitempty"`
	PaymentID   string `json:"paymentId,omitempty"`
	Reason      string `json:"reason"`
}

type DecisionInput struct {
	Decision string `json:"decision"`
	Note     string `json:"note"`
}

type Adjustment struct {
	ID            string `json:"id"`
	InvoiceID     string `json:"invoiceId"`
	PaymentID     string `json:"paymentId,omitempty"`
	Kind          Kind   `json:"kind"`
	AmountMinor   int64  `json:"amountMinor"`
	CurrencyCode  string `json:"currencyCode"`
	Reason        string `json:"reason"`
	Status        string `json:"status"`
	RequestedByID string `json:"requestedById"`
	RequestedBy   string `json:"requestedBy"`
	ReviewedBy    string `json:"reviewedBy,omitempty"`
	ReviewNote    string `json:"reviewNote,omitempty"`
	RequestedAt   string `json:"requestedAt"`
	ReviewedAt    string `json:"reviewedAt,omitempty"`
}

type InvoiceSnapshot struct {
	ID               string `json:"id"`
	ClientID         string `json:"clientId"`
	InvoiceNumber    string `json:"invoiceNumber"`
	ClientName       string `json:"clientName"`
	Status           string `json:"status"`
	CurrencyCode     string `json:"currencyCode"`
	AmountMinor      int64  `json:"amountMinor"`
	PaidMinor        int64  `json:"paidMinor"`
	OutstandingMinor int64  `json:"outstandingMinor"`
	IssuedOn         string `json:"issuedOn"`
	DueOn            string `json:"dueOn"`
	Source           string `json:"source"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
}

type DecisionResult struct {
	Adjustment Adjustment      `json:"adjustment"`
	Invoice    InvoiceSnapshot `json:"invoice"`
}
