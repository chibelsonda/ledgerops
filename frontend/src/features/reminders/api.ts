import { apiRequest } from '../../shared/api/http'
import type { DeliveryStatus, Reminder } from './types'
export const remindersApi = {
  list: async () => (await apiRequest<{ items: Reminder[] }>('/api/v1/reminders')).items,
  deliveryStatus: () => apiRequest<DeliveryStatus>('/api/v1/reminders/delivery-status'),
  generate: () => apiRequest<{ created: number }>('/api/v1/reminders/generate', { method: 'POST', successMessage: 'Overdue reminders checked successfully.' }),
  cancel: (id: string) => apiRequest<void>(`/api/v1/reminders/${id}/cancel`, { method: 'PATCH', successMessage: 'Reminder cancelled successfully.' }),
}
