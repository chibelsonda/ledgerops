import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Ban, MailCheck, RefreshCw } from 'lucide-react'
import { useAuth } from '../auth/AuthProvider'
import { formatDate, formatMoney } from '../../shared/lib/format'
import { Button } from '../../shared/ui/Button'
import { PageHeader } from '../../shared/ui/PageHeader'
import { StatusBadge } from '../../shared/ui/StatusBadge'
import { ErrorState, LoadingState } from '../../shared/ui/ViewState'
import { remindersApi } from './api'
import type { DeliveryStatus, Reminder } from './types'

function DeliveryCell({ reminder }: { reminder: Reminder }) {
  return <td>
    {reminder.deliveryEvent
      ? <StatusBadge status={reminder.deliveryEvent} />
      : <span>{reminder.attempts}/{reminder.maxAttempts} attempts</span>}
    {reminder.lastError
      ? <div className="table-subtext" title={reminder.lastError}>{reminder.lastError}</div>
      : null}
  </td>
}

function deliveryMessage(status?: DeliveryStatus) {
  if (!status?.configured) return 'SMTP is not configured, so queued reminders will not be marked as sent.'
  return `SMTP delivery is active${status.fromAddress ? ` from ${status.fromAddress}` : ''}.`
}

export function RemindersPage() {
  const { session } = useAuth()
  const queryClient = useQueryClient()
  const canEdit = session?.user.role !== 'reviewer'
  const reminders = useQuery({ queryKey: ['reminders'], queryFn: remindersApi.list })
  const deliveryStatus = useQuery({ queryKey: ['reminders', 'delivery-status'], queryFn: remindersApi.deliveryStatus })
  const generate = useMutation({
    mutationFn: remindersApi.generate,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['reminders'] }),
  })
  const cancel = useMutation({
    mutationFn: remindersApi.cancel,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['reminders'] }),
  })

  if (reminders.isLoading || deliveryStatus.isLoading) return <LoadingState label="Loading reminders" />
  if (reminders.error || deliveryStatus.error) {
    return <ErrorState message={(reminders.error ?? deliveryStatus.error)?.message ?? 'Unable to load reminders.'} />
  }

  const queued = reminders.data?.filter((item) => item.status === 'queued').length ?? 0
  return <div className="page">
    <PageHeader
      eyebrow="Collections automation"
      title="Reminder queue"
      description="Overdue invoices are queued automatically using each client’s billing policy and primary contact."
      action={canEdit
        ? <Button variant="secondary" onClick={() => generate.mutate()} disabled={generate.isPending}>
            <RefreshCw size={15} /> {generate.isPending ? 'Checking…' : 'Check overdue now'}
          </Button>
        : undefined}
    />

    <section className="panel reminder-summary">
      <MailCheck size={20} />
      <div>
        <strong>{queued} ready to send</strong>
        <span>{deliveryMessage(deliveryStatus.data)}</span>
      </div>
      {generate.data
        ? <small>{generate.data.created} new reminder{generate.data.created === 1 ? '' : 's'} queued</small>
        : null}
    </section>

    <section className="panel table-panel">
      <div className="table-scroll">
        <table>
          <thead><tr><th>Invoice</th><th>Client & recipient</th><th>Scheduled</th><th>Status</th><th>Delivery</th><th className="numeric">Outstanding</th>{canEdit ? <th><span className="sr-only">Actions</span></th> : null}</tr></thead>
          <tbody>{reminders.data?.map((item) => <tr key={item.id}>
            <td><strong>{item.invoiceNumber}</strong><div className="table-subtext">{item.subject}</div></td>
            <td><strong>{item.clientName}</strong><div className="table-subtext">{item.contactName} · {item.recipientEmail}</div></td>
            <td>{formatDate(item.scheduledFor)}</td>
            <td><StatusBadge status={item.status} /></td>
            <DeliveryCell reminder={item} />
            <td className="numeric outstanding">{formatMoney(item.outstandingMinor, item.currencyCode)}</td>
            {canEdit ? <td className="row-action">{item.status === 'queued'
              ? <Button variant="ghost" onClick={() => cancel.mutate(item.id)}><Ban size={13} /> Cancel</Button>
              : null}</td> : null}
          </tr>)}</tbody>
        </table>
      </div>
      {reminders.data?.length === 0
        ? <div className="empty-table">No reminders have been queued. Eligible overdue invoices appear here automatically.</div>
        : null}
    </section>
  </div>
}
