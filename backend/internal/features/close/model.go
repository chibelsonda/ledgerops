package close

type Actor struct {
	UserID string
	Email  string
}

type PeriodSummary struct {
	ID             string `json:"id"`
	MonthStart     string `json:"monthStart"`
	DueOn          string `json:"dueOn"`
	Status         string `json:"status"`
	CreatedByID    string `json:"createdById"`
	CreatedBy      string `json:"createdBy"`
	SubmittedByID  string `json:"submittedById,omitempty"`
	SubmittedBy    string `json:"submittedBy,omitempty"`
	ReviewedBy     string `json:"reviewedBy,omitempty"`
	ReviewNote     string `json:"reviewNote,omitempty"`
	TotalTasks     int    `json:"totalTasks"`
	CompletedTasks int    `json:"completedTasks"`
	Progress       int    `json:"progress"`
	OpenExceptions int    `json:"openExceptions"`
	SubmittedAt    string `json:"submittedAt,omitempty"`
	ReviewedAt     string `json:"reviewedAt,omitempty"`
	ClosedAt       string `json:"closedAt,omitempty"`
	CreatedAt      string `json:"createdAt"`
}

type Task struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Category     string `json:"category"`
	Position     int    `json:"position"`
	Status       string `json:"status"`
	EvidenceNote string `json:"evidenceNote,omitempty"`
	CompletedBy  string `json:"completedBy,omitempty"`
	CompletedAt  string `json:"completedAt,omitempty"`
}

type AuditEvent struct {
	ID        string         `json:"id"`
	Actor     string         `json:"actor"`
	Action    string         `json:"action"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt string         `json:"createdAt"`
}

type Detail struct {
	Period      PeriodSummary `json:"period"`
	Tasks       []Task        `json:"tasks"`
	AuditEvents []AuditEvent  `json:"auditEvents"`
}

type CreateInput struct {
	MonthStart string `json:"monthStart"`
	DueOn      string `json:"dueOn"`
}

type TaskInput struct {
	Status       string `json:"status"`
	EvidenceNote string `json:"evidenceNote"`
}

type DecisionInput struct {
	Decision string `json:"decision"`
	Note     string `json:"note"`
}
