package quickbooks

import (
	"encoding/json"
	"time"
)

const (
	EntityAccount     = "Account"
	EntityBill        = "Bill"
	EntityBillPayment = "BillPayment"
	EntityCreditMemo  = "CreditMemo"
	EntityCustomer    = "Customer"
	EntityInvoice     = "Invoice"
	EntityPayment     = "Payment"
	EntityPurchase    = "Purchase"
	EntityVendor      = "Vendor"
)

var ImportEntityTypes = []string{
	EntityAccount, EntityVendor, EntityCustomer, EntityInvoice, EntityPayment,
	EntityPurchase, EntityBill, EntityBillPayment, EntityCreditMemo,
}

type Connection struct {
	OrganizationID        string
	RealmID               string
	CompanyName           string
	AccessToken           []byte
	RefreshToken          []byte
	AccessTokenExpiresAt  time.Time
	RefreshTokenExpiresAt time.Time
	Scope                 string
	ConnectedAt           time.Time
}

type ConnectionStatus struct {
	Configured  bool              `json:"configured"`
	Connected   bool              `json:"connected"`
	Environment string            `json:"environment"`
	CompanyName string            `json:"companyName,omitempty"`
	RealmID     string            `json:"realmId,omitempty"`
	ConnectedAt *time.Time        `json:"connectedAt,omitempty"`
	Operations  OperationalStatus `json:"operations"`
}

type OperationalStatus struct {
	ScheduleEnabled    bool       `json:"scheduleEnabled"`
	NextScheduledAt    *time.Time `json:"nextScheduledAt,omitempty"`
	LastSuccessfulSync *time.Time `json:"lastSuccessfulSync,omitempty"`
	PendingWebhooks    int64      `json:"pendingWebhooks"`
	QueuedWrites       int64      `json:"queuedWrites"`
	FailedWrites       int64      `json:"failedWrites"`
}

type Authorization struct {
	AuthorizationURL string `json:"authorizationUrl"`
}

type Tokens struct {
	AccessToken           string
	RefreshToken          string
	Scope                 string
	AccessTokenExpiresIn  time.Duration
	RefreshTokenExpiresIn time.Duration
}

type Page[T any] struct {
	Items   []T
	HasMore bool
}

type CustomerRecord struct {
	ExternalID string
	Name       string
	Email      string
}

type InvoiceRecord struct {
	ExternalID         string
	CustomerExternalID string
	Number             string
	Status             string
	CurrencyCode       string
	AmountMinor        int64
	IssuedOn           time.Time
	DueOn              time.Time
	SourceUpdatedAt    *time.Time
}

type PaymentRecord struct {
	ExternalID        string
	InvoiceExternalID string
	CurrencyCode      string
	AmountMinor       int64
	ReceivedOn        time.Time
	Reference         string
}

type ExpenseRecord struct {
	ExternalID   string
	Category     string
	VendorName   string
	CurrencyCode string
	AmountMinor  int64
	IncurredOn   time.Time
}

type RemoteEntity struct {
	EntityType      string
	ExternalID      string
	SyncToken       string
	DisplayName     string
	DocumentNumber  string
	Status          string
	CurrencyCode    string
	AmountMinor     *int64
	BalanceMinor    *int64
	TransactionDate *time.Time
	SourceUpdatedAt *time.Time
	Payload         json.RawMessage
	DeletedAt       *time.Time
}

type CDCResult struct {
	Entities []RemoteEntity
	CursorAt time.Time
}
