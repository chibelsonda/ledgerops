import { describe, expect, it } from 'vitest'
import { loginSchema, passwordResetSchema, registrationSchema } from './schema'

describe('loginSchema', () => {
  it('accepts the demo controller credentials', () => {
    expect(loginSchema.safeParse({ email: 'mara@ledgerops.demo', password: 'LedgerOpsDemo!2026' }).success).toBe(true)
  })

  it('rejects invalid email and short passwords', () => {
    expect(loginSchema.safeParse({ email: 'not-an-email', password: 'short' }).success).toBe(false)
  })
})

describe('registrationSchema', () => {
  it('accepts a matching secure password', () => {
    expect(registrationSchema.safeParse({
      organizationName: 'Northstar Studio', displayName: 'Mara Chen', email: 'mara@example.com',
      password: 'correct horse battery staple', confirmPassword: 'correct horse battery staple',
    }).success).toBe(true)
  })

  it('rejects mismatched passwords', () => {
    expect(registrationSchema.safeParse({
      organizationName: 'Northstar Studio', displayName: 'Mara Chen', email: 'mara@example.com',
      password: 'correct horse battery staple', confirmPassword: 'different password value',
    }).success).toBe(false)
  })
})

describe('passwordResetSchema', () => {
  it('requires matching passwords', () => {
    expect(passwordResetSchema.safeParse({ password: 'new secure password value', confirmPassword: 'new secure password value' }).success).toBe(true)
    expect(passwordResetSchema.safeParse({ password: 'new secure password value', confirmPassword: 'another secure password' }).success).toBe(false)
  })
})
