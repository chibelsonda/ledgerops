export interface ForecastScenario {
  id: string
  name: string
  collectionRateBps: number
  collectionDelayDays: number
  expenseGrowthBps: number
  milestoneProbabilityBps: number
  createdBy: string
  createdAt: string
}

export interface ForecastAdjustment { id: string; weekStart: string; direction: 'inflow' | 'outflow'; label: string; amountMinor: number; createdBy: string; createdAt: string }
export interface ForecastWeek {
  weekStart: string
  openingBalanceMinor: number
  invoiceInflowsMinor: number
  projectInflowsMinor: number
  manualInflowsMinor: number
  operatingOutflowsMinor: number
  manualOutflowsMinor: number
  netChangeMinor: number
  endingBalanceMinor: number
}
export interface CashFlowForecast {
  scenario: ForecastScenario
  currencyCode: string
  openingCashMinor: number
  minimumCashMinor: number
  minimumCashWeek: string
  totalInflowsMinor: number
  totalOutflowsMinor: number
  weeks: ForecastWeek[]
  adjustments: ForecastAdjustment[]
}
export interface ScenarioInput { name: string; collectionRateBps: number; collectionDelayDays: number; expenseGrowthBps: number; milestoneProbabilityBps: number }
export interface AdjustmentInput { weekStart: string; direction: 'inflow' | 'outflow'; label: string; amountMinor: number }
