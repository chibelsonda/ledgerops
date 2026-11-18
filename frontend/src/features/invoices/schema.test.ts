import { describe, expect, it } from 'vitest'
import { invoiceFormSchema } from './schema'

const validInvoice = {
  clientId: '21111111-1111-4111-8111-111111111111',
  invoiceNumber: 'INV-2026-100',
  status: 'open' as const,
  currencyCode: 'USD',
  amount: 1250,
  issuedOn: '2026-11-18',
  dueOn: '2026-12-18',
  auditNote: '',
}

describe('invoiceFormSchema', () => {
  it('accepts a valid new invoice without an audit note', () => {
    expect(invoiceFormSchema(false).safeParse(validInvoice).success).toBe(true)
  })

  it('rejects a due date before the issue date', () => {
    const result = invoiceFormSchema(false).safeParse({ ...validInvoice, dueOn: '2026-11-17' })
    expect(result.success).toBe(false)
  })

  it('requires an audit note when editing', () => {
    const result = invoiceFormSchema(true).safeParse(validInvoice)
    expect(result.success).toBe(false)
  })
})
