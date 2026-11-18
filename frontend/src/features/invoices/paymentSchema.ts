import { z } from 'zod'

const paymentFieldsSchema = z.object({
  amount: z.number({ error: 'Enter an amount.' }).positive('Amount must be greater than zero.'),
  receivedOn: z.iso.date('Select a valid payment date.'),
  reference: z.string().trim().max(100, 'Keep the reference under 100 characters.'),
  auditNote: z.string().trim().min(3, 'Explain how the payment was verified.').max(500, 'Keep the note under 500 characters.'),
})

export function paymentFormSchema(outstandingAmount: number) {
  return paymentFieldsSchema.refine(
    (values) => values.amount <= outstandingAmount,
    { path: ['amount'], message: 'Payment cannot exceed the outstanding balance.' },
  )
}

export type PaymentFormValues = z.infer<typeof paymentFieldsSchema>
