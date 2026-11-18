import { useState } from 'react'
import { Button as HeadlessButton } from '@headlessui/react'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Check, ChevronRight, ShieldCheck } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { formatMoney, titleCase } from '@/shared/lib/format'
import { Button } from '@/shared/ui/Button'
import { Textarea } from '@/shared/ui/FormControls'
import { PageHeader } from '@/shared/ui/PageHeader'
import { ErrorState, LoadingState } from '@/shared/ui/ViewState'
import { reconciliationApi } from './api'
import { resolutionSchema, type ResolutionForm } from './schema'
import type { ReconciliationItem } from './types'

export function ReviewPage() {
  const queryClient = useQueryClient()
  const [selected, setSelected] = useState<ReconciliationItem | null>(null)
  const { register, handleSubmit, reset, formState: { errors } } = useForm<ResolutionForm>({
    resolver: zodResolver(resolutionSchema),
    mode: 'onChange',
    defaultValues: { note: '' },
  })
  const { data = [], isLoading, error } = useQuery({ queryKey: ['reconciliation'], queryFn: reconciliationApi.list })
  const resolveItem = useMutation({
    mutationFn: ({ id, note }: { id: string; note: string }) => reconciliationApi.resolve(id, note),
    onSuccess: async () => {
      setSelected(null)
      reset()
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['reconciliation'] }),
        queryClient.invalidateQueries({ queryKey: ['dashboard'] }),
      ])
    },
  })

  if (isLoading) return <LoadingState label="Loading review queue" />
  if (error) return <ErrorState message={error.message} />

  const openItems = data.filter((item) => item.status === 'open')
  const resolvedItems = data.filter((item) => item.status === 'resolved')
  return (
    <div className="page">
      <PageHeader eyebrow="Financial controls" title="Review queue" description="Investigate exceptions, document the decision, and leave a complete audit trail." />
      <div className="review-layout">
        <section className="panel review-list">
          <div className="panel-heading"><div><span className="eyebrow">Needs attention</span><h2>{openItems.length} open items</h2></div></div>
          {openItems.map((item) => (
            <HeadlessButton className={`review-item ${selected?.id === item.id ? 'selected' : ''}`} key={item.id} type="button" onClick={() => { setSelected(item); reset() }}>
              <span className={`severity-mark severity-${item.severity}`} />
              <span className="review-copy"><small>{titleCase(item.kind)}</small><strong>{item.title}</strong><span>{item.description}</span></span>
              {item.amountMinor !== null && item.currencyCode ? <strong className="review-amount">{formatMoney(item.amountMinor, item.currencyCode)}</strong> : null}
              <ChevronRight size={18} />
            </HeadlessButton>
          ))}
          {openItems.length === 0 ? <div className="all-clear"><ShieldCheck /><strong>Everything is reconciled</strong><span>No open exceptions remain in this workspace.</span></div> : null}
          {resolvedItems.length > 0 ? <div className="resolved-summary"><Check size={15} /> {resolvedItems.length} item{resolvedItems.length === 1 ? '' : 's'} resolved</div> : null}
        </section>

        <aside className="panel resolution-panel">
          {selected ? (
            <>
              <span className="eyebrow">Document resolution</span>
              <h2>{selected.title}</h2>
              <p>{selected.description}</p>
              <div className="detail-grid"><span>Risk</span><strong className={`severity-text-${selected.severity}`}>{titleCase(selected.severity)}</strong><span>Exception type</span><strong>{titleCase(selected.kind)}</strong></div>
              <form className="mt-5" onSubmit={handleSubmit((values) => resolveItem.mutate({ id: selected.id, note: values.note }))} noValidate>
                <Textarea label="Resolution note" rows={5} placeholder="What did you verify, and what action was taken?" error={errors.note?.message} {...register('note')} />
                {resolveItem.error ? <p className="form-error">{resolveItem.error.message}</p> : null}
                <Button className="mt-3" full type="submit" disabled={resolveItem.isPending}>
                  <Check size={16} /> {resolveItem.isPending ? 'Saving…' : 'Mark as resolved'}
                </Button>
                <small className="audit-hint">This action is written to the audit log with your note and timestamp.</small>
              </form>
            </>
          ) : (
            <div className="select-prompt"><ShieldCheck /><strong>Select an exception</strong><span>Review its context and record how it was resolved.</span></div>
          )}
        </aside>
      </div>
    </div>
  )
}
