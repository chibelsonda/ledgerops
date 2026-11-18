package invoices

const (
	DefaultPageSize = 10
	MaxPageSize     = 50
)

type Invoice struct {
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

type ListParams struct {
	Query    string
	Status   string
	Page     int
	PageSize int
}

type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	TotalItems int64 `json:"totalItems"`
	TotalPages int   `json:"totalPages"`
}

type Page struct {
	Items      []Invoice  `json:"items"`
	Pagination Pagination `json:"pagination"`
}

type ClientOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type MutationInput struct {
	ClientID      string `json:"clientId"`
	InvoiceNumber string `json:"invoiceNumber"`
	Status        string `json:"status"`
	CurrencyCode  string `json:"currencyCode"`
	AmountMinor   int64  `json:"amountMinor"`
	IssuedOn      string `json:"issuedOn"`
	DueOn         string `json:"dueOn"`
	AuditNote     string `json:"auditNote,omitempty"`
}

type AuditEvent struct {
	ID        string         `json:"id"`
	Actor     string         `json:"actor"`
	Action    string         `json:"action"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt string         `json:"createdAt"`
}

type Actor struct {
	UserID string
	Email  string
}

type PaymentInput struct {
	AmountMinor int64  `json:"amountMinor"`
	ReceivedOn  string `json:"receivedOn"`
	Reference   string `json:"reference"`
	AuditNote   string `json:"auditNote"`
}

type Payment struct {
	ID           string `json:"id"`
	InvoiceID    string `json:"invoiceId"`
	CurrencyCode string `json:"currencyCode"`
	AmountMinor  int64  `json:"amountMinor"`
	ReceivedOn   string `json:"receivedOn"`
	Reference    string `json:"reference"`
	Source       string `json:"source"`
	RecordedBy   string `json:"recordedBy"`
	Reversed     bool   `json:"reversed"`
	CreatedAt    string `json:"createdAt"`
}

type PaymentResult struct {
	Payment Payment `json:"payment"`
	Invoice Invoice `json:"invoice"`
}
