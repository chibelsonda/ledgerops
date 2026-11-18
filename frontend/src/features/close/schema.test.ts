import { describe, expect, it } from 'vitest'
import { closeDecisionSchema, closeEvidenceSchema, closePeriodSchema } from './schema'

describe('month-end close schemas', () => {
  it('accepts a valid close period', () => expect(closePeriodSchema.safeParse({ monthStart: '2026-11-01', dueOn: '2026-12-05' }).success).toBe(true))
  it('rejects a due date before the close month', () => expect(closePeriodSchema.safeParse({ monthStart: '2026-11-01', dueOn: '2026-10-31' }).success).toBe(false))
  it('requires auditable evidence and review notes', () => {
    expect(closeEvidenceSchema.safeParse({ evidenceNote: 'ok' }).success).toBe(false)
    expect(closeDecisionSchema.safeParse({ note: 'Reviewed against the close binder.' }).success).toBe(true)
  })
})
