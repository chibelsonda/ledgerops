import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Banknote, LockKeyhole } from 'lucide-react'
import { useEffect, useMemo } from 'react'
import { useForm } from 'react-hook-form'
import { formatDate, formatMoney } from '../../../shared/lib/format'
import { Button } from '../../../shared/ui/Button'
import { Input, Textarea } from '../../../shared/ui/FormControls'
import { invoicesApi } from '../api'
import { localDate } from '../formUtils'
import { paymentFormSchema, type PaymentFormValues } from '../paymentSchema'
import type { Invoice, PaymentInput } from '../types'

interface PaymentPanelProps {
  invoice: Invoice
  onRecorded: (invoice: Invoice) => void
}

function defaultValues(): PaymentFormValues {
  return { amount: 0, receivedOn: localDate(), reference: '', auditNote: '' }
}

export function PaymentPanel({ invoice, onRecorded }: PaymentPanelProps) {
  const queryClient = useQueryClient()
  const outstandingAmount = invoice.outstandingMinor / 100
  const schema = useMemo(() => paymentFormSchema(outstandingAmount), [outstandingAmount])
  const { register, handleSubmit, reset, formState: { errors } } = useForm<PaymentFormValues>({
    resolver: zodResolver(schema),
    mode: 'onChange',
    defaultValues: defaultValues(),
  })
  const paymentsQuery = useQuery({
    queryKey: ['invoices', invoice.id, 'payments'],
    queryFn: () => invoicesApi.payments(invoice.id),
  })
  const recordPayment = useMutation({
    mutationFn: (values: PaymentFormValues) => {
      const input: PaymentInput = {
        amountMinor: Math.round(values.amount * 100),
        receivedOn: values.receivedOn,
        reference: values.reference.trim(),
        auditNote: values.auditNote.trim(),
      }
      return invoicesApi.recordPayment(invoice.id, input)
    },
    onSuccess: async (result) => {
      reset(defaultValues())
      onRecorded(result.invoice)
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['invoices'] }),
        queryClient.invalidateQueries({ queryKey: ['dashboard'] }),
        queryClient.invalidateQueries({ queryKey: ['invoices', invoice.id, 'payments'] }),
        queryClient.invalidateQueries({ queryKey: ['invoices', invoice.id, 'audit-events'] }),
      ])
    },
  })

  useEffect(() => {
    reset(defaultValues())
  }, [invoice.id, reset])

  const canRecord = invoice.outstandingMinor > 0 && ['open', 'overdue', 'partial'].includes(invoice.status)

  return (
    <section className="payment-panel">
      <div className="payment-panel-heading">
        <div><Banknote size={16} /><strong>Payments</strong></div>
        <span>{formatMoney(invoice.outstandingMinor, invoice.currencyCode)} outstanding</span>
      </div>

      {paymentsQuery.isLoading ? <p className="payment-empty">Loading payments…</p> : null}
      {paymentsQuery.error ? <p className="form-error">{paymentsQuery.error.message}</p> : null}
      {paymentsQuery.data?.length === 0 ? <p className="payment-empty">No payments recorded.</p> : null}
      {paymentsQuery.data && paymentsQuery.data.length > 0 ? (
        <div className="payment-list">
          {paymentsQuery.data.map((payment) => (
            <article className={`payment-row ${payment.reversed ? 'payment-reversed' : ''}`} key={payment.id}>
              <div><strong>{formatMoney(payment.amountMinor, payment.currencyCode)}{payment.reversed ? ' · Reversed' : ''}</strong><span>{payment.reference || 'No reference'}</span></div>
              <div><strong>{formatDate(payment.receivedOn)}</strong><span>{payment.recordedBy}</span></div>
              <LockKeyhole aria-label={payment.reversed ? 'Reversed by immutable adjustment' : 'Immutable record'} size={14} />
            </article>
          ))}
        </div>
      ) : null}

      {canRecord ? (
        <form className="payment-form" onSubmit={handleSubmit((values) => recordPayment.mutate(values))} noValidate>
          <div className="payment-form-grid">
            <Input label="Payment amount" type="number" min="0.01" max={outstandingAmount} step="0.01" error={errors.amount?.message} {...register('amount', { valueAsNumber: true })} />
            <Input label="Payment date" type="date" error={errors.receivedOn?.message} {...register('receivedOn')} />
            <Input label="Payment reference" placeholder="Bank transfer or receipt ID" error={errors.reference?.message} {...register('reference')} />
          </div>
          <Textarea label="Payment audit note" rows={2} placeholder="How was this payment verified?" error={errors.auditNote?.message} {...register('auditNote')} />
          {recordPayment.error ? <p className="form-error" role="alert">{recordPayment.error.message}</p> : null}
          <div className="payment-form-action">
            <small><LockKeyhole size={12} /> Payment records cannot be edited or deleted.</small>
            <Button type="submit" disabled={recordPayment.isPending}>{recordPayment.isPending ? 'Recording…' : 'Record payment'}</Button>
          </div>
        </form>
      ) : (
        <p className="payment-locked"><LockKeyhole size={13} /> {invoice.status === 'paid' ? 'This invoice is fully paid.' : 'Payments can only be recorded against open invoices.'}</p>
      )}
    </section>
  )
}
