import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { ChevronLeft, ChevronRight, Download, Eye, Pencil, Plus, Search } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useAuth } from '../auth/AuthProvider'
import { formatDate, formatMoney } from '../../shared/lib/format'
import { Button } from '../../shared/ui/Button'
import { Input, Select } from '../../shared/ui/FormControls'
import { PageHeader } from '../../shared/ui/PageHeader'
import { StatusBadge } from '../../shared/ui/StatusBadge'
import { ErrorState, LoadingState } from '../../shared/ui/ViewState'
import { invoicesApi } from './api'
import { InvoiceDialog } from './components/InvoiceDialog'
import type { Invoice, InvoiceStatus } from './types'

function exportInvoices(invoices: Invoice[]) {
  const header = ['Invoice', 'Client', 'Issued', 'Due', 'Status', 'Amount minor', 'Outstanding minor']
  const rows = invoices.map((invoice) => [invoice.invoiceNumber, invoice.clientName, invoice.issuedOn, invoice.dueOn, invoice.status, invoice.amountMinor, invoice.outstandingMinor])
  const csv = [header, ...rows].map((row) => row.map((value) => `"${String(value).replaceAll('"', '""')}"`).join(',')).join('\n')
  const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8' }))
  const link = document.createElement('a')
  link.href = url
  link.download = 'ledgerops-invoices.csv'
  link.click()
  URL.revokeObjectURL(url)
}

export function InvoicesPage() {
  const { session } = useAuth()
  const canEdit = session?.user.role === 'admin' || session?.user.role === 'controller'
  const [searchInput, setSearchInput] = useState('')
  const [query, setQuery] = useState('')
  const [status, setStatus] = useState<InvoiceStatus | 'all'>('all')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [dialogInvoice, setDialogInvoice] = useState<Invoice | null | undefined>(undefined)

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setQuery(searchInput.trim())
      setPage(1)
    }, 300)
    return () => window.clearTimeout(timer)
  }, [searchInput])

  const filters = { query, status, page, pageSize }
  const invoicesQuery = useQuery({
    queryKey: ['invoices', filters],
    queryFn: () => invoicesApi.list(filters),
    placeholderData: keepPreviousData,
  })
  const clientsQuery = useQuery({
    queryKey: ['invoices', 'clients'],
    queryFn: invoicesApi.clients,
    enabled: canEdit,
    staleTime: 5 * 60_000,
  })

  const data = invoicesQuery.data
  const invoices = data?.items ?? []
  const pagination = data?.pagination

  useEffect(() => {
    if (pagination && pagination.totalPages > 0 && page > pagination.totalPages) {
      setPage(pagination.totalPages)
    }
  }, [page, pagination])

  if (invoicesQuery.isLoading) return <LoadingState label="Loading invoices" />
  if (invoicesQuery.error) return <ErrorState message={invoicesQuery.error.message} />

  const firstRecord = pagination && pagination.totalItems > 0 ? (pagination.page - 1) * pagination.pageSize + 1 : 0
  const lastRecord = pagination ? Math.min(pagination.page * pagination.pageSize, pagination.totalItems) : 0

  return (
    <div className="page">
      <PageHeader
        eyebrow="Accounts receivable"
        title="Invoices"
        description="Track what was billed, what was collected, and where follow-up is needed."
        action={(
          <div className="header-actions">
            <Button variant="secondary" type="button" onClick={() => exportInvoices(invoices)} disabled={invoices.length === 0}><Download size={16} /> Export page</Button>
            {canEdit ? <Button type="button" onClick={() => setDialogInvoice(null)}><Plus size={16} /> New invoice</Button> : null}
          </div>
        )}
      />
      <section className="panel table-panel">
        <div className="table-tools">
          <div className="invoice-search"><Input hideLabel label="Search invoices" icon={<Search size={16} />} value={searchInput} onChange={(event) => setSearchInput(event.target.value)} placeholder="Search client or invoice…" /></div>
          <Select hideLabel label="Filter by status" value={status} onChange={(event) => { setStatus(event.target.value as InvoiceStatus | 'all'); setPage(1) }}>
            <option value="all">All statuses</option>
            <option value="draft">Draft</option>
            <option value="open">Open</option>
            <option value="partial">Partial</option>
            <option value="overdue">Overdue</option>
            <option value="paid">Paid</option>
            <option value="void">Void</option>
          </Select>
          <span className="record-count">{invoicesQuery.isFetching ? 'Updating…' : `${firstRecord}–${lastRecord} of ${pagination?.totalItems ?? 0}`}</span>
        </div>
        <div className="table-scroll">
          <table>
            <thead><tr><th>Invoice</th><th>Client</th><th>Issued</th><th>Due</th><th>Status</th><th className="numeric">Amount</th><th className="numeric">Outstanding</th>{canEdit ? <th><span className="sr-only">Actions</span></th> : null}</tr></thead>
            <tbody>
              {invoices.map((invoice) => (
                <tr key={invoice.id}>
                  <td><strong>{invoice.invoiceNumber}</strong></td>
                  <td>{invoice.clientName}</td>
                  <td>{formatDate(invoice.issuedOn)}</td>
                  <td>{formatDate(invoice.dueOn)}</td>
                  <td><StatusBadge status={invoice.status} /></td>
                  <td className="numeric">{formatMoney(invoice.amountMinor, invoice.currencyCode)}</td>
                  <td className={`numeric ${invoice.outstandingMinor > 0 ? 'outstanding' : ''}`}>{formatMoney(invoice.outstandingMinor, invoice.currencyCode)}</td>
                  {canEdit ? <td className="row-action"><Button aria-label={`${invoice.source === 'project_billing' ? 'View' : 'Edit'} ${invoice.invoiceNumber}`} variant="ghost" type="button" onClick={() => setDialogInvoice(invoice)}>{invoice.source === 'project_billing' ? <Eye size={14} /> : <Pencil size={14} />} {invoice.source === 'project_billing' ? 'View' : 'Edit'}</Button></td> : null}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {invoices.length === 0 ? <div className="empty-table">No invoices match those filters.</div> : null}
        <div className="table-pagination">
          <div className="page-size">
            <span>Rows per page</span>
            <Select hideLabel label="Rows per page" value={pageSize} onChange={(event) => { setPageSize(Number(event.target.value)); setPage(1) }}>
              <option value="5">5</option><option value="10">10</option><option value="20">20</option>
            </Select>
          </div>
          <span>Page {pagination?.page ?? 1} of {Math.max(pagination?.totalPages ?? 0, 1)}</span>
          <Button aria-label="Previous invoice page" variant="secondary" type="button" disabled={!pagination || pagination.page <= 1 || invoicesQuery.isFetching} onClick={() => setPage((current) => current - 1)}><ChevronLeft size={15} /></Button>
          <Button aria-label="Next invoice page" variant="secondary" type="button" disabled={!pagination || pagination.page >= pagination.totalPages || invoicesQuery.isFetching} onClick={() => setPage((current) => current + 1)}><ChevronRight size={15} /></Button>
        </div>
      </section>

      {canEdit ? (
        <InvoiceDialog
          open={dialogInvoice !== undefined}
          invoice={dialogInvoice ?? null}
          clients={clientsQuery.data ?? []}
          onClose={() => setDialogInvoice(undefined)}
          onPaymentRecorded={(invoice) => setDialogInvoice(invoice)}
        />
      ) : null}
    </div>
  )
}
