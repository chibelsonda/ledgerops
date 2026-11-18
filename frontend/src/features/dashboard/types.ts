export interface Dashboard {
  organization: { id: string; name: string; baseCurrency: string }
  metrics: {
    cashPositionMinor: number
    revenueMtdMinor: number
    outstandingArMinor: number
    expensesMtdMinor: number
    openReviewCount: number
  }
  trend: Array<{ month: string; revenueMinor: number; expensesMinor: number }>
  budget: Array<{ category: string; budgetMinor: number; actualMinor: number; varianceMinor: number }>
  aging: Array<{ bucket: string; amountMinor: number }>
}

