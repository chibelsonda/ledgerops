import { describe, expect, it } from 'vitest'
import { resolutionSchema } from './schema'

describe('resolutionSchema', () => {
  it('trims and accepts a useful audit note', () => {
    expect(resolutionSchema.parse({ note: '  Confirmed payment with the client.  ' })).toEqual({
      note: 'Confirmed payment with the client.',
    })
  })

  it('rejects notes that cannot explain the decision', () => {
    expect(resolutionSchema.safeParse({ note: 'x' }).success).toBe(false)
  })
})

