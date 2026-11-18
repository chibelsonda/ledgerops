import { apiRequest } from '@/shared/api/http'
import type { ClientOption, Invoice, InvoiceAuditEvent, InvoiceFilters, InvoiceMutationInput, InvoicePage, Payment, PaymentInput, PaymentResult } from './types'

function listPath(filters: InvoiceFilters) {
  const params = new URLSearchParams({
    page: String(filters.page),
    pageSize: String(filters.pageSize),
  })
  if (filters.query) params.set('query', filters.query)
  if (filters.status !== 'all') params.set('status', filters.status)
  return `/api/v1/invoices?${params.toString()}`
}

export const invoicesApi = {
  list: (filters: InvoiceFilters) => apiRequest<InvoicePage>(listPath(filters)),
  clients: async () => (await apiRequest<{ items: ClientOption[] }>('/api/v1/invoices/clients')).items,
  create: (input: InvoiceMutationInput) => apiRequest<Invoice>('/api/v1/invoices', {
    method: 'POST',
    body: JSON.stringify(input),
    successMessage: 'Invoice created successfully.',
  }),
  update: (invoiceId: string, input: InvoiceMutationInput) => apiRequest<Invoice>(`/api/v1/invoices/${invoiceId}`, {
    method: 'PATCH',
    body: JSON.stringify(input),
    successMessage: 'Invoice saved successfully.',
  }),
  auditEvents: async (invoiceId: string) => (
    await apiRequest<{ items: InvoiceAuditEvent[] }>(`/api/v1/invoices/${invoiceId}/audit-events`)
  ).items,
  payments: async (invoiceId: string) => (
    await apiRequest<{ items: Payment[] }>(`/api/v1/invoices/${invoiceId}/payments`)
  ).items,
  recordPayment: (invoiceId: string, input: PaymentInput) => apiRequest<PaymentResult>(`/api/v1/invoices/${invoiceId}/payments`, {
    method: 'POST',
    body: JSON.stringify(input),
    successMessage: 'Payment recorded successfully.',
  }),
}
