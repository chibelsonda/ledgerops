import { apiFormRequest, apiRequest } from '@/shared/api/http'
import type { Adjustment, AdjustmentDecisionInput, AdjustmentDecisionResult, CreateAdjustmentInput } from './types'

export const adjustmentsApi = {
  list: async (invoiceId: string) => (
    await apiRequest<{ items: Adjustment[] }>(`/invoices/${invoiceId}/adjustments`)
  ).items,
  create: (invoiceId: string, input: CreateAdjustmentInput) => apiFormRequest<Adjustment>(`/invoices/${invoiceId}/adjustments`, {
    method: 'POST',
    body: JSON.stringify(input),
    successMessage: 'Adjustment submitted for approval.',
  }),
  decide: (adjustmentId: string, input: AdjustmentDecisionInput) => apiFormRequest<AdjustmentDecisionResult>(`/adjustments/${adjustmentId}`, {
    method: 'PATCH',
    body: JSON.stringify(input),
    successMessage: 'Adjustment decision saved.',
  }),
}
