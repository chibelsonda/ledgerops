import { Dialog, DialogBackdrop, DialogPanel, DialogTitle, Tab, TabGroup, TabList, TabPanel, TabPanels } from '@headlessui/react'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Clock3, Eye, FilePlus2, Pencil, X } from 'lucide-react'
import { useEffect, useMemo } from 'react'
import { useForm } from 'react-hook-form'
import { formatDate, formatMoney, formatRelativeTime, titleCase } from '../../../shared/lib/format'
import { Button } from '../../../shared/ui/Button'
import { Input, Select, Textarea } from '../../../shared/ui/FormControls'
import { AdjustmentPanel } from '../../adjustments/components/AdjustmentPanel'
import { invoicesApi } from '../api'
import { localDate } from '../formUtils'
import { invoiceFormSchema, type InvoiceFormValues } from '../schema'
import type { ClientOption, Invoice, InvoiceMutationInput } from '../types'
import { PaymentPanel } from './PaymentPanel'

interface InvoiceDialogProps {
  open: boolean
  invoice: Invoice | null
  clients: ClientOption[]
  onClose: () => void
  onPaymentRecorded: (invoice: Invoice) => void
}

function defaultValues(invoice: Invoice | null): InvoiceFormValues {
  return {
    clientId: invoice?.clientId ?? '',
    invoiceNumber: invoice?.invoiceNumber ?? '',
    status: invoice?.status ?? 'draft',
    currencyCode: invoice?.currencyCode ?? 'USD',
    amount: invoice ? invoice.amountMinor / 100 : 0,
    issuedOn: invoice?.issuedOn ?? localDate(),
    dueOn: invoice?.dueOn ?? localDate(30),
    auditNote: '',
  }
}

function mutationInput(values: InvoiceFormValues, includeAuditNote: boolean): InvoiceMutationInput {
  return {
    clientId: values.clientId,
    invoiceNumber: values.invoiceNumber.trim(),
    status: values.status,
    currencyCode: values.currencyCode.trim().toUpperCase(),
    amountMinor: Math.round(values.amount * 100),
    issuedOn: values.issuedOn,
    dueOn: values.dueOn,
    ...(includeAuditNote ? { auditNote: values.auditNote.trim() } : {}),
  }
}

export function InvoiceDialog({ open, invoice, clients, onClose, onPaymentRecorded }: InvoiceDialogProps) {
  const queryClient = useQueryClient()
  const isEditing = invoice !== null
  const isProjectBilling = invoice?.source === 'project_billing'
  const schema = useMemo(() => invoiceFormSchema(isEditing), [isEditing])
  const { register, handleSubmit, reset, formState: { errors } } = useForm<InvoiceFormValues>({
    resolver: zodResolver(schema),
    mode: 'onChange',
    defaultValues: defaultValues(invoice),
  })

  useEffect(() => {
    if (open) reset(defaultValues(invoice))
  }, [invoice, open, reset])

  const auditQuery = useQuery({
    queryKey: ['invoices', invoice?.id, 'audit-events'],
    queryFn: () => invoicesApi.auditEvents(invoice!.id),
    enabled: open && isEditing,
  })
  const saveInvoice = useMutation({
    mutationFn: (values: InvoiceFormValues) => {
      const input = mutationInput(values, isEditing)
      return invoice ? invoicesApi.update(invoice.id, input) : invoicesApi.create(input)
    },
    onSuccess: async (savedInvoice) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['invoices'] }),
        queryClient.invalidateQueries({ queryKey: ['dashboard'] }),
        queryClient.invalidateQueries({ queryKey: ['invoices', savedInvoice.id, 'audit-events'] }),
      ])
      onClose()
    },
  })

  function close() {
    if (!saveInvoice.isPending) onClose()
  }

  return (
    <Dialog className="invoice-dialog" open={open} onClose={close}>
      <DialogBackdrop transition className="invoice-dialog-backdrop" />
      <div className="invoice-dialog-positioner">
        <DialogPanel transition className="invoice-dialog-panel invoice-tabbed-panel">
          <div className="invoice-dialog-heading">
            <div className="invoice-dialog-title-icon">{isProjectBilling ? <Eye size={17} /> : isEditing ? <Pencil size={17} /> : <FilePlus2 size={18} />}</div>
            <div>
              <span className="eyebrow">{isProjectBilling ? 'Project billing' : isEditing ? 'Controlled change' : 'Accounts receivable'}</span>
              <DialogTitle>{isEditing ? `${isProjectBilling ? 'View' : 'Edit'} ${invoice.invoiceNumber}` : 'Create invoice'}</DialogTitle>
            </div>
            <Button aria-label="Close invoice form" className="invoice-dialog-close" variant="ghost" type="button" onClick={close}><X size={24} /></Button>
          </div>

          <TabGroup className="invoice-tabs">
            <TabList className="invoice-tab-list">
              <Tab className="invoice-tab">Details</Tab>
              {invoice ? <Tab className="invoice-tab">Payments</Tab> : null}
              {invoice ? <Tab className="invoice-tab">Credits &amp; reversals</Tab> : null}
              {invoice ? <Tab className="invoice-tab">Audit history</Tab> : null}
            </TabList>
            <TabPanels className="invoice-tab-panels">
              <TabPanel unmount={false} className="invoice-tab-panel">
                {isProjectBilling && invoice ? <div className="invoice-form"><p className="project-policy-note">This invoice was generated from an approved billing milestone. Its commercial fields are immutable; payments, credits, and reversals remain available through their controlled workflows.</p><div className="detail-grid"><span>Client</span><strong>{invoice.clientName}</strong><span>Status</span><strong>{titleCase(invoice.status)}</strong><span>Amount</span><strong>{formatMoney(invoice.amountMinor, invoice.currencyCode)}</strong><span>Issued</span><strong>{formatDate(invoice.issuedOn)}</strong><span>Due</span><strong>{formatDate(invoice.dueOn)}</strong></div></div> : <form className="invoice-form" onSubmit={handleSubmit((values) => saveInvoice.mutate(values))} noValidate>
                  <div className="invoice-form-grid">
                    <Input label="Invoice number" placeholder="INV-2026-100" error={errors.invoiceNumber?.message} {...register('invoiceNumber')} />
                    <Select label="Client" error={errors.clientId?.message} {...register('clientId')}>
                      <option value="">Select a client</option>
                      {clients.map((client) => <option key={client.id} value={client.id}>{client.name}</option>)}
                    </Select>
                    {invoice && (invoice.paidMinor > 0 || invoice.status === 'paid') ? (
                      <div className="derived-field">
                        <span>Status</span>
                        <strong>{titleCase(invoice.status)}</strong>
                        <small>Derived from approved ledger entries</small>
                        <input type="hidden" {...register('status')} />
                      </div>
                    ) : (
                      <Select label="Status" error={errors.status?.message} {...register('status')}>
                        <option value="draft">Draft</option>
                        <option value="open">Open</option>
                        <option value="overdue">Overdue</option>
                        <option value="void">Void</option>
                      </Select>
                    )}
                    <Input label="Amount" type="number" min="0.01" step="0.01" error={errors.amount?.message} {...register('amount', { valueAsNumber: true })} />
                    <Input label="Currency" maxLength={3} error={errors.currencyCode?.message} {...register('currencyCode')} />
                    <Input label="Issue date" type="date" error={errors.issuedOn?.message} {...register('issuedOn')} />
                    <Input label="Due date" type="date" error={errors.dueOn?.message} {...register('dueOn')} />
                  </div>

                  {isEditing ? (
                    <Textarea label="Audit note" rows={3} placeholder="Why is this invoice changing?" error={errors.auditNote?.message} {...register('auditNote')} />
                  ) : null}
                  {saveInvoice.error ? <p className="form-error" role="alert">{saveInvoice.error.message}</p> : null}

                  <div className="invoice-dialog-actions">
                    <Button variant="ghost" type="button" onClick={close}>Cancel</Button>
                    <Button type="submit" disabled={saveInvoice.isPending}>
                      {saveInvoice.isPending ? 'Saving…' : isEditing ? 'Save changes' : 'Create invoice'}
                    </Button>
                  </div>
                </form>}
              </TabPanel>

              {invoice ? <TabPanel unmount={false} className="invoice-tab-panel"><PaymentPanel invoice={invoice} onRecorded={onPaymentRecorded} /></TabPanel> : null}

              {invoice ? <TabPanel unmount={false} className="invoice-tab-panel"><AdjustmentPanel invoice={invoice} onInvoiceChanged={onPaymentRecorded} /></TabPanel> : null}

              {isEditing ? (
                <TabPanel unmount={false} className="invoice-tab-panel">
                  <section className="invoice-audit">
                    <div className="invoice-audit-heading"><Clock3 size={15} /><strong>Audit history</strong></div>
                    {auditQuery.isLoading ? <p>Loading activity…</p> : null}
                    {auditQuery.error ? <p className="form-error">{auditQuery.error.message}</p> : null}
                    {auditQuery.data?.length === 0 ? <p>No recorded changes yet.</p> : null}
                    {auditQuery.data?.map((event) => {
                      const note = typeof event.metadata.note === 'string' ? event.metadata.note : null
                      const changes = event.metadata.changes && typeof event.metadata.changes === 'object'
                        ? Object.keys(event.metadata.changes).length
                        : 0
                      return (
                        <article className="invoice-audit-event" key={event.id}>
                          <span>{titleCase(event.action.replaceAll('.', '_').replace('invoice_', ''))}</span>
                          <strong>{event.actor}</strong>
                          <small>{formatRelativeTime(event.createdAt)}{changes > 0 ? ` · ${changes} field${changes === 1 ? '' : 's'} changed` : ''}</small>
                          {note ? <p>{note}</p> : null}
                        </article>
                      )
                    })}
                  </section>
                </TabPanel>
              ) : null}
            </TabPanels>
          </TabGroup>
        </DialogPanel>
      </div>
    </Dialog>
  )
}
