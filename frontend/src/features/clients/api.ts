import { apiFormRequest, apiRequest } from '@/shared/api/http'
import type { Client, ClientInput, ClientPage, Contact, ContactInput } from './types'

export const clientsApi = {
  list: (query: string, page: number, pageSize: number) => apiRequest<ClientPage>(`/clients?${new URLSearchParams({ query, page: String(page), pageSize: String(pageSize) })}`),
  create: (input: ClientInput) => apiFormRequest<Client>('/clients', { method: 'POST', body: JSON.stringify(input), successMessage: 'Client created successfully.' }),
  update: (id: string, input: ClientInput) => apiFormRequest<Client>(`/clients/${id}`, { method: 'PATCH', body: JSON.stringify(input), successMessage: 'Client saved successfully.' }),
  contacts: async (clientId: string) => (await apiRequest<{ items: Contact[] }>(`/clients/${clientId}/contacts`)).items,
  createContact: (clientId: string, input: ContactInput) => apiFormRequest<Contact>(`/clients/${clientId}/contacts`, { method: 'POST', body: JSON.stringify(input), successMessage: 'Contact added successfully.' }),
  updateContact: (contactId: string, input: ContactInput) => apiFormRequest<Contact>(`/contacts/${contactId}`, { method: 'PATCH', body: JSON.stringify(input), successMessage: 'Contact updated successfully.' }),
}
