import { describe, expect, it } from 'vitest'
import { clientFormSchema, contactFormSchema } from './schema'

describe('client validation', () => {
  it('requires a valid billing contact', () => {
    const result = clientFormSchema(false).safeParse({ name: 'Acme', paymentTermsDays: 30, remindersEnabled: true, reminderDelayDays: 3, contactName: 'A', contactEmail: 'bad', contactPhone: '', contactRole: '', auditNote: '' })
    expect(result.success).toBe(false)
  })
  it('accepts a primary contact', () => {
    expect(contactFormSchema.safeParse({ contactName: 'Avery Stone', contactEmail: 'avery@example.com', contactPhone: '', contactRole: 'Finance', isPrimary: true }).success).toBe(true)
  })
})
