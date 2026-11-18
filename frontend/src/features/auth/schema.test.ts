import { describe, expect, it } from 'vitest'
import { loginSchema } from './schema'

describe('loginSchema', () => {
  it('accepts the demo controller credentials', () => {
    expect(loginSchema.safeParse({ email: 'mara@ledgerops.demo', password: 'LedgerOpsDemo!2026' }).success).toBe(true)
  })

  it('rejects invalid email and short passwords', () => {
    expect(loginSchema.safeParse({ email: 'not-an-email', password: 'short' }).success).toBe(false)
  })
})
