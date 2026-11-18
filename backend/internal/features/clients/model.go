package clients

const DefaultPageSize = 10

type Client struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	PaymentTermsDays  int    `json:"paymentTermsDays"`
	RemindersEnabled  bool   `json:"remindersEnabled"`
	ReminderDelayDays int    `json:"reminderDelayDays"`
	PrimaryContact    string `json:"primaryContact"`
	PrimaryEmail      string `json:"primaryEmail"`
	ContactCount      int    `json:"contactCount"`
	InvoiceCount      int    `json:"invoiceCount"`
	OutstandingMinor  int64  `json:"outstandingMinor"`
	CurrencyCode      string `json:"currencyCode"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
}

type Contact struct {
	ID        string `json:"id"`
	ClientID  string `json:"clientId"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	RoleTitle string `json:"roleTitle"`
	IsPrimary bool   `json:"isPrimary"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type ListParams struct {
	Query    string
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
	Items      []Client   `json:"items"`
	Pagination Pagination `json:"pagination"`
}

type PrimaryContactInput struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	RoleTitle string `json:"roleTitle"`
}

type MutationInput struct {
	Name              string               `json:"name"`
	PaymentTermsDays  int                  `json:"paymentTermsDays"`
	RemindersEnabled  bool                 `json:"remindersEnabled"`
	ReminderDelayDays int                  `json:"reminderDelayDays"`
	PrimaryContact    *PrimaryContactInput `json:"primaryContact,omitempty"`
	AuditNote         string               `json:"auditNote,omitempty"`
}

type ContactInput struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	RoleTitle string `json:"roleTitle"`
	IsPrimary bool   `json:"isPrimary"`
}

type Actor struct {
	UserID string
	Email  string
}
