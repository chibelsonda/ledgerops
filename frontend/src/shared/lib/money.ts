const MINOR_UNITS_PER_MAJOR = 100

export function toMinorUnits(amount: number): number {
  if (!Number.isFinite(amount) || Math.abs(amount) > Number.MAX_SAFE_INTEGER / MINOR_UNITS_PER_MAJOR) {
    throw new RangeError('Amount is outside the supported range.')
  }
  return Math.round((amount + Math.sign(amount) * Number.EPSILON) * MINOR_UNITS_PER_MAJOR)
}

export function fromMinorUnits(amountMinor: number): number {
  if (!Number.isSafeInteger(amountMinor)) throw new RangeError('Minor-unit amount must be a safe integer.')
  return amountMinor / MINOR_UNITS_PER_MAJOR
}
