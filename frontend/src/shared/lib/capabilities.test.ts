import { describe, expect, it } from 'vitest'
import { capabilities } from './capabilities'

describe('role capabilities', () => {
  it('keeps operator and approval policies explicit', () => {
    expect(capabilities.operate('controller')).toBe(true)
    expect(capabilities.operate('reviewer')).toBe(false)
    expect(capabilities.administer('admin')).toBe(true)
    expect(capabilities.administer('controller')).toBe(false)
    expect(capabilities.createClosePeriod('controller')).toBe(true)
    expect(capabilities.createClosePeriod('admin')).toBe(false)
  })
})
