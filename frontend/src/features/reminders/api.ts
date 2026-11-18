import { apiRequest } from '@/shared/api/http'
import type { DeliveryStatus, Reminder } from './types'
export const remindersApi = {
  list: async () => (await apiRequest<{ items: Reminder[] }>('/reminders')).items,
  deliveryStatus: () => apiRequest<DeliveryStatus>('/reminders/delivery-status'),
  generate: () => apiRequest<{ created: number }>('/reminders/generate', { method: 'POST', successMessage: 'Overdue reminders checked successfully.' }),
  cancel: (id: string) => apiRequest<void>(`/reminders/${id}/cancel`, { method: 'PATCH', successMessage: 'Reminder cancelled successfully.' }),
}
