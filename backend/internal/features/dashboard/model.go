package dashboard

type View struct {
	Organization Organization     `json:"organization"`
	Metrics      Metrics          `json:"metrics"`
	Trend        []TrendPoint     `json:"trend"`
	Budget       []BudgetCategory `json:"budget"`
	Aging        []AgingBucket    `json:"aging"`
}

type Organization struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	BaseCurrency string `json:"baseCurrency"`
}

type Metrics struct {
	CashPositionMinor  int64 `json:"cashPositionMinor"`
	RevenueMTDMinor    int64 `json:"revenueMtdMinor"`
	OutstandingARMinor int64 `json:"outstandingArMinor"`
	ExpensesMTDMinor   int64 `json:"expensesMtdMinor"`
	OpenReviewCount    int   `json:"openReviewCount"`
}

type TrendPoint struct {
	Month         string `json:"month"`
	RevenueMinor  int64  `json:"revenueMinor"`
	ExpensesMinor int64  `json:"expensesMinor"`
}

type BudgetCategory struct {
	Category      string `json:"category"`
	BudgetMinor   int64  `json:"budgetMinor"`
	ActualMinor   int64  `json:"actualMinor"`
	VarianceMinor int64  `json:"varianceMinor"`
}

type AgingBucket struct {
	Bucket      string `json:"bucket"`
	AmountMinor int64  `json:"amountMinor"`
}
