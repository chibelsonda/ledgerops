package sync

type Run struct {
	ID               string           `json:"id"`
	Provider         string           `json:"provider"`
	Status           string           `json:"status"`
	RecordsProcessed int              `json:"recordsProcessed"`
	ErrorMessage     *string          `json:"errorMessage"`
	CreatedAt        string           `json:"createdAt"`
	FinishedAt       *string          `json:"finishedAt"`
	Entities         []EntityProgress `json:"entities" gorm:"-"`
}

type EntityProgress struct {
	EntityType       string  `json:"entityType"`
	Status           string  `json:"status"`
	RecordsProcessed int     `json:"recordsProcessed"`
	ErrorMessage     *string `json:"errorMessage"`
}

type Job struct {
	ID             string
	OrganizationID string
	SyncRunID      string
	Provider       string
	Attempts       int
}
