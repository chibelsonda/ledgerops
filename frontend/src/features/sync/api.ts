import { apiRequest } from '@/shared/api/http'
import type { QuickBooksAuthorization, QuickBooksConnection, SyncRun } from './types'

export const syncApi = {
  list: async () => (await apiRequest<{ items: SyncRun[] }>('/api/v1/sync-runs')).items,
  enqueue: () => apiRequest<SyncRun>('/api/v1/sync-runs', { method: 'POST', successMessage: 'Synchronization started.' }),
}

export const quickBooksApi = {
  status: () => apiRequest<QuickBooksConnection>('/api/v1/integrations/quickbooks'),
  connect: () => apiRequest<QuickBooksAuthorization>('/api/v1/integrations/quickbooks/connect', { method: 'POST', successMessage: 'QuickBooks authorization started.' }),
  disconnect: () => apiRequest<void>('/api/v1/integrations/quickbooks', { method: 'DELETE', successMessage: 'QuickBooks disconnected successfully.' }),
}
