import { describe, expect, it } from 'vitest'
import { paymentFormSchema } from './paymentSchema'

const payment = {
  amount: 300,
  receivedOn: '2026-11-18',
  reference: 'BANK-123',
  auditNote: 'Confirmed against the bank deposit.',
}

describe('paymentFormSchema', () => {
  it('accepts a payment within the outstanding balance', () => {
    expect(paymentFormSchema(1_000).safeParse(payment).success).toBe(true)
  })

  it('rejects overpayment', () => {
    expect(paymentFormSchema(250).safeParse(payment).success).toBe(false)
  })

  it('requires an audit note', () => {
    expect(paymentFormSchema(1_000).safeParse({ ...payment, auditNote: '' }).success).toBe(false)
  })
})
