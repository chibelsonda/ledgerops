export type InvoiceStatus = 'draft' | 'open' | 'partial' | 'paid' | 'void' | 'overdue'

export interface Invoice {
  id: string
  clientId: string
  invoiceNumber: string
  clientName: string
  status: InvoiceStatus
  currencyCode: string
  amountMinor: number
  paidMinor: number
  outstandingMinor: number
  issuedOn: string
  dueOn: string
  source: string
  createdAt: string
  updatedAt: string
}

export interface InvoicePagination {
  page: number
  pageSize: number
  totalItems: number
  totalPages: number
}

export interface InvoicePage {
  items: Invoice[]
  pagination: InvoicePagination
}

export interface InvoiceFilters {
  query: string
  status: InvoiceStatus | 'all'
  page: number
  pageSize: number
}

export interface ClientOption {
  id: string
  name: string
}

export interface InvoiceMutationInput {
  clientId: string
  invoiceNumber: string
  status: InvoiceStatus
  currencyCode: string
  amountMinor: number
  issuedOn: string
  dueOn: string
  auditNote?: string
}

export interface InvoiceAuditEvent {
  id: string
  actor: string
  action: 'invoice.created' | 'invoice.updated' | 'invoice.created_from_milestone' | 'payment.recorded' | 'adjustment.requested' | 'adjustment.approved' | 'adjustment.rejected'
  metadata: Record<string, unknown>
  createdAt: string
}

export interface Payment {
  id: string
  invoiceId: string
  currencyCode: string
  amountMinor: number
  receivedOn: string
  reference: string
  source: string
  recordedBy: string
  reversed: boolean
  createdAt: string
}

export interface PaymentInput {
  amountMinor: number
  receivedOn: string
  reference: string
  auditNote: string
}

export interface PaymentResult {
  payment: Payment
  invoice: Invoice
}
