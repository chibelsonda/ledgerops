import { describe, expect, it } from 'vitest'
import { formatMoney, titleCase } from './format'

describe('formatMoney', () => {
  it('formats minor units without floating-point business calculations', () => {
    expect(formatMoney(825_000, 'USD')).toBe('$8,250.00')
  })
})

describe('titleCase', () => {
  it('makes machine statuses readable', () => {
    expect(titleCase('partial_payment')).toBe('Partial Payment')
  })
})

