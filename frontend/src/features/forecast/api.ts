import { apiRequest } from '@/shared/api/http'
import type { AdjustmentInput, CashFlowForecast, ForecastScenario, ScenarioInput } from './types'

export const forecastApi = {
  scenarios: async () => (await apiRequest<{ items: ForecastScenario[] }>('/cash-flow/scenarios')).items,
  get: (scenarioId: string) => apiRequest<CashFlowForecast>(`/cash-flow/scenarios/${scenarioId}`),
  createScenario: (input: ScenarioInput) => apiRequest<CashFlowForecast>('/cash-flow/scenarios', { method: 'POST', body: JSON.stringify(input), successMessage: 'Forecast scenario created.' }),
  addAdjustment: (scenarioId: string, input: AdjustmentInput) => apiRequest<CashFlowForecast>(`/cash-flow/scenarios/${scenarioId}/adjustments`, { method: 'POST', body: JSON.stringify(input), successMessage: 'Forecast adjustment added.' }),
}
