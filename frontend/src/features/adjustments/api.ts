import { apiRequest } from '@/shared/api/http'
import type { Adjustment, AdjustmentDecisionInput, AdjustmentDecisionResult, CreateAdjustmentInput } from './types'

export const adjustmentsApi = {
  list: async (invoiceId: string) => (
    await apiRequest<{ items: Adjustment[] }>(`/api/v1/invoices/${invoiceId}/adjustments`)
  ).items,
  create: (invoiceId: string, input: CreateAdjustmentInput) => apiRequest<Adjustment>(`/api/v1/invoices/${invoiceId}/adjustments`, {
    method: 'POST',
    body: JSON.stringify(input),
    successMessage: 'Adjustment submitted for approval.',
  }),
  decide: (adjustmentId: string, input: AdjustmentDecisionInput) => apiRequest<AdjustmentDecisionResult>(`/api/v1/adjustments/${adjustmentId}`, {
    method: 'PATCH',
    body: JSON.stringify(input),
    successMessage: 'Adjustment decision saved.',
  }),
}
