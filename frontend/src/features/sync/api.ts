import { apiRequest } from '@/shared/api/http'
import type { QuickBooksAuthorization, QuickBooksConnection, SyncRun } from './types'

export const syncApi = {
  list: async () => (await apiRequest<{ items: SyncRun[] }>('/sync-runs')).items,
  enqueue: () => apiRequest<SyncRun>('/sync-runs', { method: 'POST', successMessage: 'Synchronization started.' }),
}

export const quickBooksApi = {
  status: () => apiRequest<QuickBooksConnection>('/integrations/quickbooks'),
  connect: () => apiRequest<QuickBooksAuthorization>('/integrations/quickbooks/connect', { method: 'POST', successMessage: 'QuickBooks authorization started.' }),
  disconnect: () => apiRequest<void>('/integrations/quickbooks', { method: 'DELETE', successMessage: 'QuickBooks disconnected successfully.' }),
}
