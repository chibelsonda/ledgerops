import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { BadgeCheck, FileWarning, LockKeyhole } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useAuth } from '@/features/auth/AuthProvider'
import { invoicesApi } from '@/features/invoices/api'
import type { Invoice } from '@/features/invoices/types'
import { formatMoney, formatRelativeTime, titleCase } from '@/shared/lib/format'
import { toMinorUnits } from '@/shared/lib/money'
import { capabilities } from '@/shared/lib/capabilities'
import { queryKeys } from '@/shared/api/queryKeys'
import { Button } from '@/shared/ui/Button'
import { Input, Select, Textarea } from '@/shared/ui/FormControls'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { adjustmentsApi } from '@/features/adjustments/api'
import { adjustmentDecisionSchema, adjustmentFormSchema, type AdjustmentDecisionValues, type AdjustmentFormValues } from '@/features/adjustments/schema'
import type { Adjustment, AdjustmentDecisionInput, CreateAdjustmentInput } from '@/features/adjustments/types'

interface AdjustmentPanelProps {
  invoice: Invoice
  onInvoiceChanged: (invoice: Invoice) => void
}

const createDefaults: AdjustmentFormValues = { kind: 'credit_note', amount: 0, reason: '' }

export function AdjustmentPanel({ invoice, onInvoiceChanged }: AdjustmentPanelProps) {
  const { session } = useAuth()
  const queryClient = useQueryClient()
  const [decisionTarget, setDecisionTarget] = useState<Adjustment | null>(null)
  const [decision, setDecision] = useState<'approved' | 'rejected'>('approved')
  const schema = useMemo(() => adjustmentFormSchema(invoice.outstandingMinor / 100), [invoice.outstandingMinor])
  const createForm = useForm<AdjustmentFormValues>({ resolver: zodResolver(schema), mode: 'onChange', defaultValues: createDefaults })
  const decisionForm = useForm<AdjustmentDecisionValues>({ resolver: zodResolver(adjustmentDecisionSchema), mode: 'onChange', defaultValues: { note: '' } })
  const kind = createForm.watch('kind')
  const adjustmentsQuery = useQuery({ queryKey: queryKeys.invoices.adjustments(invoice.id), queryFn: () => adjustmentsApi.list(invoice.id) })
  const paymentsQuery = useQuery({ queryKey: queryKeys.invoices.payments(invoice.id), queryFn: () => invoicesApi.payments(invoice.id) })
  const blockedPaymentIds = new Set(adjustmentsQuery.data?.filter((item) => item.kind === 'payment_reversal' && item.status !== 'rejected').map((item) => item.paymentId))
  const reversiblePayments = paymentsQuery.data?.filter((payment) => !payment.reversed && !blockedPaymentIds.has(payment.id)) ?? []

  const refresh = async () => Promise.all([
    queryClient.invalidateQueries({ queryKey: queryKeys.invoices.all }),
    queryClient.invalidateQueries({ queryKey: queryKeys.dashboard }),
    queryClient.invalidateQueries({ queryKey: queryKeys.invoices.adjustments(invoice.id) }),
    queryClient.invalidateQueries({ queryKey: queryKeys.invoices.payments(invoice.id) }),
    queryClient.invalidateQueries({ queryKey: queryKeys.invoices.audit(invoice.id) }),
  ])

  const createAdjustment = useMutation({
    mutationFn: (values: AdjustmentFormValues) => {
      const input: CreateAdjustmentInput = values.kind === 'credit_note'
        ? { kind: values.kind, amountMinor: toMinorUnits(values.amount), reason: values.reason.trim() }
        : { kind: values.kind, paymentId: values.paymentId, reason: values.reason.trim() }
      return adjustmentsApi.create(invoice.id, input)
    },
    onSuccess: async () => {
      createForm.reset(createDefaults)
      await refresh()
    },
  })
  const decideAdjustment = useMutation({
    mutationFn: (values: AdjustmentDecisionValues) => adjustmentsApi.decide(decisionTarget!.id, { decision, note: values.note.trim() } satisfies AdjustmentDecisionInput),
    onSuccess: async (result) => {
      setDecisionTarget(null)
      decisionForm.reset({ note: '' })
      onInvoiceChanged(result.invoice)
      await refresh()
    },
  })

  useEffect(() => {
    createForm.reset(createDefaults)
    setDecisionTarget(null)
    decisionForm.reset({ note: '' })
  }, [createForm, decisionForm, invoice.id])

  const openDecision = (adjustment: Adjustment, value: 'approved' | 'rejected') => {
    setDecisionTarget(adjustment)
    setDecision(value)
    decisionForm.reset({ note: '' })
  }

  return (
    <section className="adjustment-panel">
      <div className="adjustment-heading">
        <div><FileWarning size={16} /><strong>Credits & reversals</strong></div>
        <span>Approval-controlled ledger</span>
      </div>

      {adjustmentsQuery.isLoading ? <p className="payment-empty">Loading adjustments…</p> : null}
      {adjustmentsQuery.error ? <p className="form-error">{adjustmentsQuery.error.message}</p> : null}
      {adjustmentsQuery.data?.length === 0 ? <p className="payment-empty">No adjustment requests.</p> : null}
      {adjustmentsQuery.data?.map((adjustment) => {
        const canReview = capabilities.administer(session?.user.role) && adjustment.status === 'pending' && adjustment.requestedById !== session?.user.userId
        return (
          <article className="adjustment-row" key={adjustment.id}>
            <div className="adjustment-row-summary">
              <div><strong>{titleCase(adjustment.kind)}</strong><StatusBadge status={adjustment.status} /></div>
              <strong>{formatMoney(adjustment.amountMinor, adjustment.currencyCode)}</strong>
              <small>{adjustment.requestedBy} · {formatRelativeTime(adjustment.requestedAt)}</small>
              <p>{adjustment.reason}</p>
              {adjustment.reviewNote ? <p className="adjustment-review-note"><BadgeCheck size={12} /> {adjustment.reviewedBy}: {adjustment.reviewNote}</p> : null}
            </div>
            {canReview ? (
              <div className="adjustment-actions">
                <Button type="button" variant="secondary" onClick={() => openDecision(adjustment, 'rejected')}>Reject</Button>
                <Button type="button" onClick={() => openDecision(adjustment, 'approved')}>Approve</Button>
              </div>
            ) : null}
            {capabilities.administer(session?.user.role) && adjustment.status === 'pending' && adjustment.requestedById === session?.user.userId ? <small className="separation-note">Another admin must review your request.</small> : null}
          </article>
        )
      })}

      {decisionTarget ? (
        <form className="adjustment-decision" onSubmit={decisionForm.handleSubmit((values) => decideAdjustment.mutate(values))} noValidate>
          <Textarea label={`${titleCase(decision)} review note`} rows={2} placeholder="Document the review evidence and decision." error={decisionForm.formState.errors.note?.message} {...decisionForm.register('note')} />
          {decideAdjustment.error ? <p className="form-error" role="alert">{decideAdjustment.error.message}</p> : null}
          <div className="adjustment-actions">
            <Button type="button" variant="ghost" onClick={() => setDecisionTarget(null)}>Cancel</Button>
            <Button type="submit" disabled={decideAdjustment.isPending}>{decideAdjustment.isPending ? 'Saving…' : `Confirm ${decision}`}</Button>
          </div>
        </form>
      ) : null}

      {invoice.status !== 'draft' && invoice.status !== 'void' ? (
        <form className="adjustment-form" onSubmit={createForm.handleSubmit((values) => createAdjustment.mutate(values))} noValidate>
          <Select label="Adjustment type" error={createForm.formState.errors.kind?.message} {...createForm.register('kind')}>
            <option value="credit_note">Credit note</option>
            <option value="payment_reversal">Payment reversal</option>
          </Select>
          {kind === 'credit_note' ? (
            <Input label="Credit amount" type="number" min="0.01" max={invoice.outstandingMinor / 100} step="0.01" error={createForm.formState.errors.amount?.message} {...createForm.register('amount', { valueAsNumber: true })} />
          ) : (
            <Select label="Payment to reverse" error={createForm.formState.errors.paymentId?.message} {...createForm.register('paymentId')}>
              <option value="">Select a payment</option>
              {reversiblePayments.map((payment) => <option key={payment.id} value={payment.id}>{formatMoney(payment.amountMinor, payment.currencyCode)} · {payment.reference || payment.receivedOn}</option>)}
            </Select>
          )}
          <Textarea label="Required justification" rows={2} placeholder="Why is this accounting adjustment necessary?" error={createForm.formState.errors.reason?.message} {...createForm.register('reason')} />
          {createAdjustment.error ? <p className="form-error" role="alert">{createAdjustment.error.message}</p> : null}
          <div className="payment-form-action">
            <small><LockKeyhole size={12} /> Approved entries are immutable.</small>
            <Button type="submit" disabled={createAdjustment.isPending}>{createAdjustment.isPending ? 'Submitting…' : 'Submit for approval'}</Button>
          </div>
        </form>
      ) : <p className="payment-locked"><LockKeyhole size={13} /> Adjustments require an issued invoice.</p>}
    </section>
  )
}
