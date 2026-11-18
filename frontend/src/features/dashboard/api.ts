import { apiRequest } from '@/shared/api/http'
import type { Dashboard } from './types'

export const dashboardApi = {
  get: () => apiRequest<Dashboard>('/api/v1/dashboard'),
}

