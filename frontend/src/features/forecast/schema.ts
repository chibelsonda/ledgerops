import { z } from 'zod'

export const scenarioSchema = z.object({
  name: z.string().trim().min(2).max(80),
  collectionRate: z.number().min(0).max(100),
  collectionDelayDays: z.number().int().min(0).max(180),
  expenseGrowth: z.number().min(-100).max(500),
  milestoneProbability: z.number().min(0).max(100),
})

export const adjustmentSchema = z.object({
  weekStart: z.string().regex(/^\d{4}-\d{2}-\d{2}$/),
  direction: z.enum(['inflow', 'outflow']),
  label: z.string().trim().min(2).max(120),
  amount: z.number().positive().max(1_000_000_000_000),
})

export type ScenarioForm = z.infer<typeof scenarioSchema>
export type AdjustmentForm = z.infer<typeof adjustmentSchema>
