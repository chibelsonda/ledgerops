package sync

type Run struct {
	ID               string  `json:"id"`
	Provider         string  `json:"provider"`
	Status           string  `json:"status"`
	RecordsProcessed int     `json:"recordsProcessed"`
	ErrorMessage     *string `json:"errorMessage"`
	CreatedAt        string  `json:"createdAt"`
	FinishedAt       *string `json:"finishedAt"`
}

type Job struct {
	ID             string
	OrganizationID string
	SyncRunID      string
	Provider       string
	Attempts       int
}
