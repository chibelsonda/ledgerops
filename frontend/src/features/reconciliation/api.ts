import { apiRequest } from '@/shared/api/http'
import type { ReconciliationItem } from './types'

export const reconciliationApi = {
  list: async () => (await apiRequest<{ items: ReconciliationItem[] }>('/api/v1/reconciliation')).items,
  resolve: (itemId: string, note: string) => apiRequest<void>(`/api/v1/reconciliation/${itemId}`, {
    method: 'PATCH',
    body: JSON.stringify({ note }),
    successMessage: 'Review item resolved successfully.',
  }),
}
