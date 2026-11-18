import type { Invoice } from '@/features/invoices/types'

export type AdjustmentKind = 'credit_note' | 'payment_reversal'
export type AdjustmentStatus = 'pending' | 'approved' | 'rejected'

export interface Adjustment {
  id: string
  invoiceId: string
  paymentId?: string
  kind: AdjustmentKind
  amountMinor: number
  currencyCode: string
  reason: string
  status: AdjustmentStatus
  requestedById: string
  requestedBy: string
  reviewedBy?: string
  reviewNote?: string
  requestedAt: string
  reviewedAt?: string
}

export interface CreateAdjustmentInput {
  kind: AdjustmentKind
  amountMinor?: number
  paymentId?: string
  reason: string
}

export interface AdjustmentDecisionInput {
  decision: 'approved' | 'rejected'
  note: string
}

export interface AdjustmentDecisionResult {
  adjustment: Adjustment
  invoice: Invoice
}
