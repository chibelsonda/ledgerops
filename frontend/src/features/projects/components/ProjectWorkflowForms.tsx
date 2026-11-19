import type { FieldValues, UseFormReturn } from 'react-hook-form'
import { formatMoney } from '@/shared/lib/format'
import { Button } from '@/shared/ui/Button'
import { Input, Textarea } from '@/shared/ui/FormControls'
import { Modal } from '@/shared/ui/Modal'
import type { ChangeOrderFormValues, DecisionFormValues, InvoiceMilestoneFormValues, MilestoneFormValues } from '@/features/projects/schema'
import type { BillingMilestone } from '@/features/projects/types'

interface FormStateProps<T extends FieldValues> {
  form: UseFormReturn<T>
  pending: boolean
  error: Error | null
  onCancel: () => void
}

export function ChangeOrderForm({ form, pending, error, onCancel, onSubmit }: FormStateProps<ChangeOrderFormValues> & { onSubmit: (values: ChangeOrderFormValues) => void }) {
  return <form className="project-inline-form" onSubmit={form.handleSubmit(onSubmit)} noValidate>
    <Input label="Change-order number" error={form.formState.errors.changeOrderNumber?.message} {...form.register('changeOrderNumber')} />
    <Input label="Title" error={form.formState.errors.title?.message} {...form.register('title')} />
    <Input label="Amount (+/−)" type="number" step="0.01" error={form.formState.errors.amount?.message} {...form.register('amount', { valueAsNumber: true })} />
    <Textarea className="col-span-full" label="Scope justification" rows={3} error={form.formState.errors.description?.message} {...form.register('description')} />
    {error ? <p className="form-error col-span-full">{error.message}</p> : null}
    <div className="invoice-dialog-actions col-span-full"><Button type="button" variant="ghost" onClick={onCancel}>Cancel</Button><Button type="submit" disabled={pending}>Submit for approval</Button></div>
  </form>
}

export function MilestoneForm({ form, pending, error, onCancel, onSubmit }: FormStateProps<MilestoneFormValues> & { onSubmit: (values: MilestoneFormValues) => void }) {
  return <form className="project-inline-form" onSubmit={form.handleSubmit(onSubmit)} noValidate>
    <Input label="Milestone name" error={form.formState.errors.name?.message} {...form.register('name')} />
    <Input label="Amount" type="number" min="0.01" step="0.01" error={form.formState.errors.amount?.message} {...form.register('amount', { valueAsNumber: true })} />
    <Input label="Due date" type="date" error={form.formState.errors.dueOn?.message} {...form.register('dueOn')} />
    {error ? <p className="form-error col-span-full">{error.message}</p> : null}
    <div className="invoice-dialog-actions col-span-full"><Button type="button" variant="ghost" onClick={onCancel}>Cancel</Button><Button type="submit" disabled={pending}>Add milestone</Button></div>
  </form>
}

export function DecisionDialog({ label, form, open, pending, error, onClose, onDecision }: { label: string; form: UseFormReturn<DecisionFormValues>; open: boolean; pending: boolean; error: Error | null; onClose: () => void; onDecision: (decision: 'approved' | 'rejected', values: DecisionFormValues) => void }) {
  return <Modal open={open} title={`Review ${label}`} onClose={onClose}>
    <p>Approval is final and creates an immutable financial record.</p>
    <Textarea label="Review note" rows={4} error={form.formState.errors.note?.message} {...form.register('note')} />
    {error ? <p className="form-error">{error.message}</p> : null}
    <div className="invoice-dialog-actions"><Button variant="ghost" onClick={onClose}>Cancel</Button><Button variant="secondary" disabled={pending} onClick={() => void form.handleSubmit((values) => onDecision('rejected', values))()}>Reject</Button><Button disabled={pending} onClick={() => void form.handleSubmit((values) => onDecision('approved', values))()}>Approve</Button></div>
  </Modal>
}

export function MilestoneInvoiceDialog({ target, currencyCode, form, pending, error, onClose, onSubmit }: { target: BillingMilestone | null; currencyCode?: string; form: UseFormReturn<InvoiceMilestoneFormValues>; pending: boolean; error: Error | null; onClose: () => void; onSubmit: (values: InvoiceMilestoneFormValues) => void }) {
  return <Modal open={Boolean(target)} title={`Invoice ${target?.name ?? ''}`} onClose={onClose}>
    <p>The invoice amount is fixed at {target && currencyCode ? formatMoney(target.amountMinor, currencyCode) : ''}.</p>
    <form onSubmit={form.handleSubmit(onSubmit)} noValidate>
      <div className="project-invoice-grid"><Input label="Invoice number" error={form.formState.errors.invoiceNumber?.message} {...form.register('invoiceNumber')} /><Input label="Issue date" type="date" error={form.formState.errors.issuedOn?.message} {...form.register('issuedOn')} /><Input label="Due date" type="date" error={form.formState.errors.dueOn?.message} {...form.register('dueOn')} /></div>
      {error ? <p className="form-error">{error.message}</p> : null}
      <div className="invoice-dialog-actions"><Button type="button" variant="ghost" onClick={onClose}>Cancel</Button><Button type="submit" disabled={pending}>Create invoice</Button></div>
    </form>
  </Modal>
}
