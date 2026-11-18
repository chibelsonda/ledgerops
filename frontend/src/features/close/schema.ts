import { z } from 'zod'

const date = /^\d{4}-\d{2}-\d{2}$/

export const closePeriodSchema = z.object({
  monthStart: z.string().regex(date, 'Select a close month.'),
  dueOn: z.string().regex(date, 'Select a due date.'),
}).refine((value) => value.dueOn >= value.monthStart, { path: ['dueOn'], message: 'Due date must be on or after the close month.' })

export const closeEvidenceSchema = z.object({ evidenceNote: z.string().trim().min(3, 'Evidence must contain at least 3 characters.').max(500) })
export const closeDecisionSchema = z.object({ note: z.string().trim().min(3, 'Review note must contain at least 3 characters.').max(500) })
export type ClosePeriodForm = z.infer<typeof closePeriodSchema>
export type CloseEvidenceForm = z.infer<typeof closeEvidenceSchema>
export type CloseDecisionForm = z.infer<typeof closeDecisionSchema>
