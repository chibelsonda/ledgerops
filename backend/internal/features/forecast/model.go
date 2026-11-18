package forecast

type Actor struct {
	UserID string
	Email  string
}

type Scenario struct {
	ID                      string `json:"id"`
	Name                    string `json:"name"`
	CollectionRateBPS       int    `json:"collectionRateBps"`
	CollectionDelayDays     int    `json:"collectionDelayDays"`
	ExpenseGrowthBPS        int    `json:"expenseGrowthBps"`
	MilestoneProbabilityBPS int    `json:"milestoneProbabilityBps"`
	CreatedBy               string `json:"createdBy"`
	CreatedAt               string `json:"createdAt"`
}

type Adjustment struct {
	ID          string `json:"id"`
	WeekStart   string `json:"weekStart"`
	Direction   string `json:"direction"`
	Label       string `json:"label"`
	AmountMinor int64  `json:"amountMinor"`
	CreatedBy   string `json:"createdBy"`
	CreatedAt   string `json:"createdAt"`
}

type Week struct {
	WeekStart              string `json:"weekStart"`
	OpeningBalanceMinor    int64  `json:"openingBalanceMinor"`
	InvoiceInflowsMinor    int64  `json:"invoiceInflowsMinor"`
	ProjectInflowsMinor    int64  `json:"projectInflowsMinor"`
	ManualInflowsMinor     int64  `json:"manualInflowsMinor"`
	OperatingOutflowsMinor int64  `json:"operatingOutflowsMinor"`
	ManualOutflowsMinor    int64  `json:"manualOutflowsMinor"`
	NetChangeMinor         int64  `json:"netChangeMinor"`
	EndingBalanceMinor     int64  `json:"endingBalanceMinor"`
}

type Forecast struct {
	Scenario           Scenario     `json:"scenario"`
	CurrencyCode       string       `json:"currencyCode"`
	OpeningCashMinor   int64        `json:"openingCashMinor"`
	MinimumCashMinor   int64        `json:"minimumCashMinor"`
	MinimumCashWeek    string       `json:"minimumCashWeek"`
	TotalInflowsMinor  int64        `json:"totalInflowsMinor"`
	TotalOutflowsMinor int64        `json:"totalOutflowsMinor"`
	Weeks              []Week       `json:"weeks"`
	Adjustments        []Adjustment `json:"adjustments"`
}

type ScenarioInput struct {
	Name                    string `json:"name"`
	CollectionRateBPS       int    `json:"collectionRateBps"`
	CollectionDelayDays     int    `json:"collectionDelayDays"`
	ExpenseGrowthBPS        int    `json:"expenseGrowthBps"`
	MilestoneProbabilityBPS int    `json:"milestoneProbabilityBps"`
}

type AdjustmentInput struct {
	WeekStart   string `json:"weekStart"`
	Direction   string `json:"direction"`
	Label       string `json:"label"`
	AmountMinor int64  `json:"amountMinor"`
}
