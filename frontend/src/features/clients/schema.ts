import { z } from 'zod'

const contactFields = {
  contactName: z.string().trim().min(2, 'Enter the contact name.').max(100),
  contactEmail: z.email('Enter a valid email address.'),
  contactPhone: z.string().trim().max(40),
  contactRole: z.string().trim().max(80),
}
export const clientFormSchema = (editing: boolean) => z.object({
  name: z.string().trim().min(2, 'Enter the client name.').max(120),
  paymentTermsDays: z.number().int().min(0).max(365),
  remindersEnabled: z.boolean(),
  reminderDelayDays: z.number().int().min(0).max(90),
  ...contactFields,
  auditNote: editing ? z.string().trim().min(3, 'Explain this client change.').max(500) : z.string().optional(),
})
export const contactFormSchema = z.object({ ...contactFields, isPrimary: z.boolean() })
export type ClientFormValues = z.infer<ReturnType<typeof clientFormSchema>>
export type ContactFormValues = z.infer<typeof contactFormSchema>
