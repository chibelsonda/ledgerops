import { apiRequest } from '../../shared/api/http'
import type { CloseDecisionInput, CloseDetail, ClosePeriodSummary, CreateClosePeriodInput, UpdateCloseTaskInput } from './types'

export const closeApi = {
  list: async () => (await apiRequest<{ items: ClosePeriodSummary[] }>('/api/v1/close-periods')).items,
  get: (periodId: string) => apiRequest<CloseDetail>(`/api/v1/close-periods/${periodId}`),
  create: (input: CreateClosePeriodInput) => apiRequest<CloseDetail>('/api/v1/close-periods', { method: 'POST', body: JSON.stringify(input), successMessage: 'Close period created.' }),
  updateTask: (taskId: string, input: UpdateCloseTaskInput) => apiRequest<CloseDetail>(`/api/v1/close-tasks/${taskId}`, { method: 'PATCH', body: JSON.stringify(input), successMessage: input.status === 'completed' ? 'Close task completed.' : 'Close task reopened.' }),
  submit: (periodId: string) => apiRequest<CloseDetail>(`/api/v1/close-periods/${periodId}/submit`, { method: 'POST', body: '{}', successMessage: 'Close submitted for review.' }),
  decide: (periodId: string, input: CloseDecisionInput) => apiRequest<CloseDetail>(`/api/v1/close-periods/${periodId}/decision`, { method: 'PATCH', body: JSON.stringify(input), successMessage: input.decision === 'approved' ? 'Accounting period closed.' : 'Close returned for changes.' }),
}
