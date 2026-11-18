import { apiRequest } from '@/shared/api/http'
import type { Client, ClientInput, ClientPage, Contact, ContactInput } from './types'

export const clientsApi = {
  list: (query: string, page: number, pageSize: number) => apiRequest<ClientPage>(`/api/v1/clients?${new URLSearchParams({ query, page: String(page), pageSize: String(pageSize) })}`),
  create: (input: ClientInput) => apiRequest<Client>('/api/v1/clients', { method: 'POST', body: JSON.stringify(input), successMessage: 'Client created successfully.' }),
  update: (id: string, input: ClientInput) => apiRequest<Client>(`/api/v1/clients/${id}`, { method: 'PATCH', body: JSON.stringify(input), successMessage: 'Client saved successfully.' }),
  contacts: async (clientId: string) => (await apiRequest<{ items: Contact[] }>(`/api/v1/clients/${clientId}/contacts`)).items,
  createContact: (clientId: string, input: ContactInput) => apiRequest<Contact>(`/api/v1/clients/${clientId}/contacts`, { method: 'POST', body: JSON.stringify(input), successMessage: 'Contact added successfully.' }),
  updateContact: (contactId: string, input: ContactInput) => apiRequest<Contact>(`/api/v1/contacts/${contactId}`, { method: 'PATCH', body: JSON.stringify(input), successMessage: 'Contact updated successfully.' }),
}
