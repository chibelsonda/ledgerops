import { describe, expect, it } from 'vitest'
import { fromMinorUnits, toMinorUnits } from './money'

describe('money conversion', () => {
  it('converts decimal amounts without losing cents', () => {
    expect(toMinorUnits(10.1)).toBe(1010)
    expect(toMinorUnits(-10.1)).toBe(-1010)
    expect(fromMinorUnits(1010)).toBe(10.1)
  })

  it('rejects non-finite and unsafe amounts', () => {
    expect(() => toMinorUnits(Number.NaN)).toThrow(RangeError)
    expect(() => fromMinorUnits(Number.MAX_SAFE_INTEGER + 1)).toThrow(RangeError)
  })
})
