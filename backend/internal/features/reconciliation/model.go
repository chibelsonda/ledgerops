package reconciliation

type Item struct {
	ID           string  `json:"id"`
	Kind         string  `json:"kind"`
	Severity     string  `json:"severity"`
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	AmountMinor  *int64  `json:"amountMinor"`
	CurrencyCode *string `json:"currencyCode"`
	Status       string  `json:"status"`
	CreatedAt    string  `json:"createdAt"`
}

type Resolution struct {
	Note string `json:"note"`
}
