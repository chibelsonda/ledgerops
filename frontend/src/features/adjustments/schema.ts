import { z } from 'zod'

export const adjustmentFormSchema = (outstandingAmount: number) => z.discriminatedUnion('kind', [
  z.object({
    kind: z.literal('credit_note'),
    amount: z.number({ error: 'Enter a credit amount.' })
      .positive('Credit amount must be greater than zero.')
      .max(outstandingAmount, 'Credit cannot exceed the outstanding balance.'),
    paymentId: z.string().optional(),
    reason: z.string().trim().min(3, 'Explain why this credit is needed.').max(500, 'Keep the justification under 500 characters.'),
  }),
  z.object({
    kind: z.literal('payment_reversal'),
    amount: z.number().optional(),
    paymentId: z.uuid('Select a payment to reverse.'),
    reason: z.string().trim().min(3, 'Explain why this reversal is needed.').max(500, 'Keep the justification under 500 characters.'),
  }),
])

export const adjustmentDecisionSchema = z.object({
  note: z.string().trim().min(3, 'Add a review note.').max(500, 'Keep the review note under 500 characters.'),
})

export type AdjustmentFormValues = z.infer<ReturnType<typeof adjustmentFormSchema>>
export type AdjustmentDecisionValues = z.infer<typeof adjustmentDecisionSchema>
