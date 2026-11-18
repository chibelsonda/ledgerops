import { describe, expect, it } from 'vitest'
import { changeOrderFormSchema, invoiceMilestoneFormSchema, projectFormSchema, toMinor } from './schema'

describe('project billing schemas', () => {
  it('rejects a project ending before it starts', () => {
    const result = projectFormSchema.safeParse({ clientId: '11111111-1111-4111-8111-111111111111', projectCode: 'P-1', name: 'Launch', currencyCode: 'USD', startOn: '2026-06-10', endOn: '2026-06-09', contractNumber: 'C-1', contractTitle: 'Launch contract', originalValue: 1000 })
    expect(result.success).toBe(false)
  })

  it('supports positive and negative non-zero change orders', () => {
    const base = { changeOrderNumber: 'CO-1', title: 'Scope', description: 'Updated scope' }
    expect(changeOrderFormSchema.safeParse({ ...base, amount: 250 }).success).toBe(true)
    expect(changeOrderFormSchema.safeParse({ ...base, amount: -250 }).success).toBe(true)
    expect(changeOrderFormSchema.safeParse({ ...base, amount: 0 }).success).toBe(false)
  })

  it('validates invoice date order and converts money to minor units', () => {
    expect(invoiceMilestoneFormSchema.safeParse({ invoiceNumber: 'INV-1', issuedOn: '2026-06-10', dueOn: '2026-06-09' }).success).toBe(false)
    expect(toMinor(1234.56)).toBe(123456)
  })
})
