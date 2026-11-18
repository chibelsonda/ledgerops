package projects

type Actor struct {
	UserID string
	Email  string
}

type ClientOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ProjectSummary struct {
	ID                   string `json:"id"`
	ProjectCode          string `json:"projectCode"`
	Name                 string `json:"name"`
	ClientID             string `json:"clientId"`
	ClientName           string `json:"clientName"`
	Status               string `json:"status"`
	CurrencyCode         string `json:"currencyCode"`
	StartOn              string `json:"startOn"`
	EndOn                string `json:"endOn,omitempty"`
	ContractID           string `json:"contractId"`
	ContractNumber       string `json:"contractNumber"`
	ContractStatus       string `json:"contractStatus"`
	OriginalValueMinor   int64  `json:"originalValueMinor"`
	ApprovedChangesMinor int64  `json:"approvedChangesMinor"`
	RevisedValueMinor    int64  `json:"revisedValueMinor"`
	AllocatedMinor       int64  `json:"allocatedMinor"`
	BilledMinor          int64  `json:"billedMinor"`
	UnallocatedMinor     int64  `json:"unallocatedMinor"`
	CreatedAt            string `json:"createdAt"`
}

type Contract struct {
	ID                 string `json:"id"`
	ContractNumber     string `json:"contractNumber"`
	Title              string `json:"title"`
	OriginalValueMinor int64  `json:"originalValueMinor"`
	Status             string `json:"status"`
	RequestedByID      string `json:"requestedById"`
	RequestedBy        string `json:"requestedBy"`
	ReviewedBy         string `json:"reviewedBy,omitempty"`
	ReviewNote         string `json:"reviewNote,omitempty"`
	RequestedAt        string `json:"requestedAt"`
	ReviewedAt         string `json:"reviewedAt,omitempty"`
}

type ChangeOrder struct {
	ID                string `json:"id"`
	ChangeOrderNumber string `json:"changeOrderNumber"`
	Title             string `json:"title"`
	Description       string `json:"description"`
	AmountMinor       int64  `json:"amountMinor"`
	Status            string `json:"status"`
	RequestedByID     string `json:"requestedById"`
	RequestedBy       string `json:"requestedBy"`
	ReviewedBy        string `json:"reviewedBy,omitempty"`
	ReviewNote        string `json:"reviewNote,omitempty"`
	RequestedAt       string `json:"requestedAt"`
	ReviewedAt        string `json:"reviewedAt,omitempty"`
}

type BillingMilestone struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	AmountMinor   int64  `json:"amountMinor"`
	DueOn         string `json:"dueOn"`
	Status        string `json:"status"`
	InvoiceID     string `json:"invoiceId,omitempty"`
	InvoiceNumber string `json:"invoiceNumber,omitempty"`
	CreatedBy     string `json:"createdBy"`
	CreatedAt     string `json:"createdAt"`
	InvoicedAt    string `json:"invoicedAt,omitempty"`
}

type AuditEvent struct {
	ID        string         `json:"id"`
	Actor     string         `json:"actor"`
	Action    string         `json:"action"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt string         `json:"createdAt"`
}

type Detail struct {
	Project      ProjectSummary     `json:"project"`
	Contract     Contract           `json:"contract"`
	ChangeOrders []ChangeOrder      `json:"changeOrders"`
	Milestones   []BillingMilestone `json:"milestones"`
	AuditEvents  []AuditEvent       `json:"auditEvents"`
}

type CreateInput struct {
	ClientID           string `json:"clientId"`
	ProjectCode        string `json:"projectCode"`
	Name               string `json:"name"`
	CurrencyCode       string `json:"currencyCode"`
	StartOn            string `json:"startOn"`
	EndOn              string `json:"endOn,omitempty"`
	ContractNumber     string `json:"contractNumber"`
	ContractTitle      string `json:"contractTitle"`
	OriginalValueMinor int64  `json:"originalValueMinor"`
}

type DecisionInput struct {
	Decision string `json:"decision"`
	Note     string `json:"note"`
}

type ChangeOrderInput struct {
	ChangeOrderNumber string `json:"changeOrderNumber"`
	Title             string `json:"title"`
	Description       string `json:"description"`
	AmountMinor       int64  `json:"amountMinor"`
}

type MilestoneInput struct {
	Name        string `json:"name"`
	AmountMinor int64  `json:"amountMinor"`
	DueOn       string `json:"dueOn"`
}

type InvoiceInput struct {
	InvoiceNumber string `json:"invoiceNumber"`
	IssuedOn      string `json:"issuedOn"`
	DueOn         string `json:"dueOn"`
}
