package reminders

import "encoding/json"

type Reminder struct {
	ID                string  `json:"id"`
	InvoiceID         string  `json:"invoiceId"`
	InvoiceNumber     string  `json:"invoiceNumber"`
	ClientID          string  `json:"clientId"`
	ClientName        string  `json:"clientName"`
	ContactName       string  `json:"contactName"`
	RecipientEmail    string  `json:"recipientEmail"`
	Status            string  `json:"status"`
	ScheduledFor      string  `json:"scheduledFor"`
	Subject           string  `json:"subject"`
	Message           string  `json:"message"`
	OutstandingMinor  int64   `json:"outstandingMinor"`
	CurrencyCode      string  `json:"currencyCode"`
	CreatedAt         string  `json:"createdAt"`
	Attempts          int     `json:"attempts"`
	MaxAttempts       int     `json:"maxAttempts"`
	LastError         *string `json:"lastError"`
	ProviderMessageID *string `json:"providerMessageId"`
	DeliveryEvent     *string `json:"deliveryEvent"`
}

type GenerateResult struct {
	Created int64 `json:"created"`
}

type Actor struct{ UserID, Email string }

type Delivery struct {
	ID             string
	RecipientEmail string
	Subject        string
	Message        string
	Attempts       int
	MaxAttempts    int
}

type EmailMessage struct {
	To        string
	Subject   string
	Body      string
	MessageID string
}

type DeliveryStatus struct {
	Configured  bool   `json:"configured"`
	FromAddress string `json:"fromAddress,omitempty"`
}

type DeliveryEventInput struct {
	EventID           string          `json:"eventId"`
	ProviderMessageID string          `json:"messageId"`
	EventType         string          `json:"event"`
	OccurredAt        string          `json:"occurredAt"`
	Payload           json.RawMessage `json:"payload,omitempty"`
}
