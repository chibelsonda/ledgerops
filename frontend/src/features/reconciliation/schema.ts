import { z } from 'zod'

export const resolutionSchema = z.object({
  note: z.string().trim().min(3, 'Add at least 3 characters.').max(500, 'Keep the note under 500 characters.'),
})

export type ResolutionForm = z.infer<typeof resolutionSchema>

