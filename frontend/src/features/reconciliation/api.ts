import { apiFormRequest, apiRequest } from '@/shared/api/http'
import type { ReconciliationItem } from './types'

export const reconciliationApi = {
  list: async () => (await apiRequest<{ items: ReconciliationItem[] }>('/reconciliation')).items,
  resolve: (itemId: string, note: string) => apiFormRequest<void>(`/reconciliation/${itemId}`, {
    method: 'PATCH',
    body: JSON.stringify({ note }),
    successMessage: 'Review item resolved successfully.',
  }),
}
