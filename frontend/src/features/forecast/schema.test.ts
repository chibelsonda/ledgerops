import { describe, expect, it } from 'vitest'
import { adjustmentSchema, scenarioSchema } from './schema'

describe('cash-flow forecast schemas', () => {
  it('accepts explicit scenario assumptions', () => expect(scenarioSchema.safeParse({ name: 'Base', collectionRate: 85, collectionDelayDays: 7, expenseGrowth: 0, milestoneProbability: 70 }).success).toBe(true))
  it('rejects impossible probabilities', () => expect(scenarioSchema.safeParse({ name: 'Bad', collectionRate: 101, collectionDelayDays: 7, expenseGrowth: 0, milestoneProbability: 70 }).success).toBe(false))
  it('requires a positive adjustment', () => expect(adjustmentSchema.safeParse({ weekStart: '2026-11-16', direction: 'outflow', label: 'Insurance', amount: 0 }).success).toBe(false))
})
