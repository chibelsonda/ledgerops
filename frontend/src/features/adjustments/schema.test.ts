import { describe, expect, it } from 'vitest'
import { adjustmentDecisionSchema, adjustmentFormSchema } from './schema'

describe('adjustment validation', () => {
  it('rejects credits above the outstanding balance', () => {
    expect(adjustmentFormSchema(100).safeParse({ kind: 'credit_note', amount: 101, reason: 'Client concession' }).success).toBe(false)
  })

  it('requires a payment for reversals', () => {
    expect(adjustmentFormSchema(100).safeParse({ kind: 'payment_reversal', paymentId: '', reason: 'Duplicate receipt' }).success).toBe(false)
  })

  it('requires an auditable review note', () => {
    expect(adjustmentDecisionSchema.safeParse({ note: 'no' }).success).toBe(false)
  })
})
