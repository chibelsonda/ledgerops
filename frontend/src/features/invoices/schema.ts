import { z } from 'zod'

const invoiceFieldsSchema = z.object({
  clientId: z.string().uuid('Select a client.'),
  invoiceNumber: z.string().trim().min(2, 'Use at least 2 characters.').max(40, 'Keep the number under 40 characters.'),
  status: z.enum(['draft', 'open', 'partial', 'paid', 'void', 'overdue']),
  currencyCode: z.string().trim().length(3, 'Use a three-letter currency code.').regex(/^[A-Za-z]{3}$/, 'Use letters only.'),
  amount: z.number({ error: 'Enter an amount.' }).positive('Amount must be greater than zero.').max(1_000_000_000_000, 'Amount is too large.'),
  issuedOn: z.iso.date('Select a valid issue date.'),
  dueOn: z.iso.date('Select a valid due date.'),
  auditNote: z.string().trim().max(500, 'Keep the note under 500 characters.'),
})

export function invoiceFormSchema(requireAuditNote: boolean) {
  return invoiceFieldsSchema.superRefine((values, context) => {
    if (values.dueOn < values.issuedOn) {
      context.addIssue({ code: 'custom', path: ['dueOn'], message: 'Due date cannot be before the issue date.' })
    }
    if (requireAuditNote && values.auditNote.length < 3) {
      context.addIssue({ code: 'custom', path: ['auditNote'], message: 'Explain the change in at least 3 characters.' })
    }
  })
}

export type InvoiceFormValues = z.infer<typeof invoiceFieldsSchema>
