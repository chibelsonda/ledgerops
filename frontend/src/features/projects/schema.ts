import { z } from 'zod'

const dateField = z.string().regex(/^\d{4}-\d{2}-\d{2}$/, 'Enter a valid date.')
const moneyField = z.number().finite().positive('Enter an amount greater than zero.').max(1_000_000_000_000)

export const projectFormSchema = z.object({
  clientId: z.uuid('Select a client.'),
  projectCode: z.string().trim().min(2, 'Enter a project code.').max(30),
  name: z.string().trim().min(2, 'Enter a project name.').max(150),
  currencyCode: z.string().trim().length(3, 'Use a three-letter currency code.'),
  startOn: dateField,
  endOn: z.union([dateField, z.literal('')]),
  contractNumber: z.string().trim().min(2, 'Enter a contract number.').max(40),
  contractTitle: z.string().trim().min(2, 'Enter a contract title.').max(150),
  originalValue: moneyField,
}).refine((value) => !value.endOn || value.endOn >= value.startOn, { path: ['endOn'], message: 'End date cannot be before the start date.' })

export const decisionFormSchema = z.object({
  note: z.string().trim().min(3, 'Enter a review note.').max(500),
})

export const changeOrderFormSchema = z.object({
  changeOrderNumber: z.string().trim().min(2, 'Enter a change-order number.').max(40),
  title: z.string().trim().min(2, 'Enter a title.').max(150),
  description: z.string().trim().min(3, 'Explain the scope change.').max(1000),
  amount: z.number().finite().refine((value) => value !== 0, 'Amount cannot be zero.').refine((value) => Math.abs(value) <= 1_000_000_000_000, 'Amount is outside the supported range.'),
})

export const milestoneFormSchema = z.object({
  name: z.string().trim().min(2, 'Enter a milestone name.').max(150),
  amount: moneyField,
  dueOn: dateField,
})

export const invoiceMilestoneFormSchema = z.object({
  invoiceNumber: z.string().trim().min(2, 'Enter an invoice number.').max(40),
  issuedOn: dateField,
  dueOn: dateField,
}).refine((value) => value.dueOn >= value.issuedOn, { path: ['dueOn'], message: 'Due date cannot be before the issue date.' })

export type ProjectFormValues = z.infer<typeof projectFormSchema>
export type DecisionFormValues = z.infer<typeof decisionFormSchema>
export type ChangeOrderFormValues = z.infer<typeof changeOrderFormSchema>
export type MilestoneFormValues = z.infer<typeof milestoneFormSchema>
export type InvoiceMilestoneFormValues = z.infer<typeof invoiceMilestoneFormSchema>

export const toMinor = (amount: number) => Math.round(amount * 100)
