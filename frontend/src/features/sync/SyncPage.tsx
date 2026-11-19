import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AlertCircle, CheckCircle2, CloudCog, Database, ExternalLink, Link2, Link2Off, RefreshCw, ShieldCheck } from 'lucide-react'
import { useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'
import { formatRelativeTime, titleCase } from '@/shared/lib/format'
import { queryKeys } from '@/shared/api/queryKeys'
import { Button } from '@/shared/ui/Button'
import { PageHeader } from '@/shared/ui/PageHeader'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { ErrorState, LoadingState } from '@/shared/ui/ViewState'
import { quickBooksApi, syncApi } from './api'

export function SyncPage() {
  const queryClient = useQueryClient()
  const [searchParams, setSearchParams] = useSearchParams()
  const callbackResult = searchParams.get('quickbooks')
  const callbackMessage = searchParams.get('message')

  const runs = useQuery({
    queryKey: queryKeys.syncRuns,
    queryFn: syncApi.list,
    refetchInterval: (query) => query.state.data?.some((run) => run.status === 'queued' || run.status === 'running') ? 1_500 : false,
  })
  const connection = useQuery({ queryKey: queryKeys.quickBooks, queryFn: quickBooksApi.status })

  useEffect(() => {
    if (callbackResult === 'connected') {
      void queryClient.invalidateQueries({ queryKey: queryKeys.quickBooks })
    }
  }, [callbackResult, queryClient])

  const enqueue = useMutation({
    mutationFn: syncApi.enqueue,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.syncRuns }),
  })
  const connect = useMutation({
    mutationFn: quickBooksApi.connect,
    onSuccess: ({ authorizationUrl }) => window.location.assign(authorizationUrl),
  })
  const disconnect = useMutation({
    mutationFn: quickBooksApi.disconnect,
    onSuccess: () => {
      setSearchParams({}, { replace: true })
      void queryClient.invalidateQueries({ queryKey: queryKeys.quickBooks })
    },
  })

  if (runs.isLoading || connection.isLoading) return <LoadingState label="Loading synchronization workspace" />
  const error = runs.error ?? connection.error
  if (error) return <ErrorState message={error.message} />

  const history = runs.data ?? []
  const source = connection.data!
  const isBusy = enqueue.isPending || history.some((run) => run.status === 'queued' || run.status === 'running')
  const connectionError = connect.error ?? disconnect.error

  return (
    <div className="page">
      <PageHeader
        eyebrow="Connected systems"
        title="Data synchronization"
        description="Connect a real QuickBooks sandbox, then monitor imports, retries, and source freshness from one auditable workflow."
        action={(
          <Button type="button" disabled={!source.connected || isBusy} onClick={() => enqueue.mutate()}>
            <RefreshCw size={16} className={isBusy ? 'spin' : ''} />
            {isBusy ? 'Sync in progress' : 'Run sandbox sync'}
          </Button>
        )}
      />

      {callbackResult && (
        <div className={`integration-notice ${callbackResult === 'connected' ? 'integration-notice-success' : 'integration-notice-error'}`} role="status">
          {callbackResult === 'connected' ? <CheckCircle2 size={17} /> : <AlertCircle size={17} />}
          <span>{callbackResult === 'connected' ? 'QuickBooks sandbox connected successfully.' : callbackMessage ?? 'QuickBooks could not be connected.'}</span>
          <Button variant="ghost" type="button" onClick={() => setSearchParams({}, { replace: true })}>Dismiss</Button>
        </div>
      )}

      <section className="integration-card">
        <div className="integration-icon">QB</div>
        <div className="integration-copy">
          <span className="eyebrow">Accounting source</span>
          <h2>{source.companyName ?? 'QuickBooks Online sandbox'}</h2>
          <p>
            {source.connected
              ? `Verified sandbox company · Realm ${source.realmId}`
              : source.configured
                ? 'Ready to authorize invoices, payments, customers, and expenses.'
                : 'Add sandbox credentials to the API environment before connecting.'}
          </p>
        </div>
        <div className="integration-actions">
          <span className={`connection-state ${source.connected ? 'is-connected' : 'is-disconnected'}`}>
            <i /> {source.connected ? 'Connected' : source.configured ? 'Not connected' : 'Setup required'}
          </span>
          {source.connected ? (
            <Button variant="ghost" type="button" disabled={disconnect.isPending} onClick={() => disconnect.mutate()}>
              <Link2Off size={15} /> {disconnect.isPending ? 'Disconnecting' : 'Disconnect'}
            </Button>
          ) : (
            <Button type="button" disabled={!source.configured || connect.isPending} onClick={() => connect.mutate()}>
              {connect.isPending ? <RefreshCw size={15} className="spin" /> : <Link2 size={15} />}
              {connect.isPending ? 'Opening QuickBooks' : 'Connect QuickBooks'}
              {!connect.isPending && <ExternalLink size={13} />}
            </Button>
          )}
        </div>
      </section>
      {connectionError && <p className="integration-inline-error" role="alert">{connectionError.message}</p>}

      <section className="sync-principles">
        <div><Database /><strong>Idempotent upserts</strong><span>External IDs prevent duplicate financial records.</span></div>
        <div><CloudCog /><strong>Background processing</strong><span>PostgreSQL queue with locking, retries, and backoff.</span></div>
        <div><ShieldCheck /><strong>Protected credentials</strong><span>One-time OAuth state and encrypted token storage.</span></div>
      </section>
      <section className="panel table-panel">
        <div className="panel-heading"><div><span className="eyebrow">Operations</span><h2>Recent sync runs</h2></div></div>
        <div className="table-scroll">
          <table>
            <thead><tr><th>Source</th><th>Started</th><th>Status</th><th className="numeric">Records</th><th>Result</th></tr></thead>
            <tbody>{history.map((run) => (
              <tr key={run.id}><td><strong>{titleCase(run.provider)}</strong></td><td>{formatRelativeTime(run.createdAt)}</td><td><StatusBadge status={run.status} /></td><td className="numeric">{run.recordsProcessed}</td><td>{run.status === 'succeeded' ? <span className="success-copy"><CheckCircle2 size={15} /> Completed without errors</span> : run.errorMessage ?? 'Waiting for worker'}</td></tr>
            ))}</tbody>
          </table>
          {history.length === 0 && <div className="empty-table">No synchronization runs yet.</div>}
        </div>
      </section>
    </div>
  )
}
