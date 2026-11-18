import { useQuery } from '@tanstack/react-query'
import { Banknote, CircleDollarSign, Clock3, ReceiptText, TrendingUp } from 'lucide-react'
import { Area, AreaChart, Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import { formatMoney } from '../../shared/lib/format'
import { PageHeader } from '../../shared/ui/PageHeader'
import { ErrorState, LoadingState } from '../../shared/ui/ViewState'
import { dashboardApi } from './api'
import { MetricCard } from './components/MetricCard'

const chartMoney = (value: number) => formatMoney(value, 'USD', true)

export function OverviewPage() {
  const { data, isLoading, error } = useQuery({ queryKey: ['dashboard'], queryFn: dashboardApi.get })

  if (isLoading) return <LoadingState />
  if (error || !data) return <ErrorState message={error?.message ?? 'No dashboard data was returned.'} />

  const { metrics, trend, budget, aging, organization } = data
  return (
    <div className="page">
      <PageHeader
        eyebrow="Financial command center"
        title="The numbers that need your attention."
        description="A reliable view of cash, receivables, and operating spend—updated from your accounting workflow."
        action={<div className="as-of"><Clock3 size={15} /> As of today</div>}
      />

      <section className="metrics-grid" aria-label="Financial summary">
        <MetricCard label="Cash position" value={formatMoney(metrics.cashPositionMinor, organization.baseCurrency)} detail="Payments less recorded spend" icon={CircleDollarSign} />
        <MetricCard label="Revenue this month" value={formatMoney(metrics.revenueMtdMinor, organization.baseCurrency)} detail="Booked from issued invoices" icon={TrendingUp} />
        <MetricCard label="Outstanding A/R" value={formatMoney(metrics.outstandingArMinor, organization.baseCurrency)} detail="Across open and overdue invoices" icon={ReceiptText} tone="warning" />
        <MetricCard label="Operating spend" value={formatMoney(metrics.expensesMtdMinor, organization.baseCurrency)} detail="Month to date" icon={Banknote} />
      </section>

      <section className="dashboard-grid">
        <article className="panel trend-panel">
          <div className="panel-heading">
            <div><span className="eyebrow">Performance</span><h2>Revenue & operating spend</h2></div>
            <div className="legend"><span><i className="dot revenue" /> Revenue</span><span><i className="dot expense" /> Spend</span></div>
          </div>
          <div className="chart-wrap">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={trend} margin={{ top: 12, right: 8, left: 0, bottom: 0 }}>
                <defs>
                  <linearGradient id="revenueFill" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stopColor="#1f7550" stopOpacity={0.24} /><stop offset="100%" stopColor="#1f7550" stopOpacity={0} /></linearGradient>
                </defs>
                <CartesianGrid stroke="#e8e5dc" vertical={false} strokeDasharray="3 3" />
                <XAxis dataKey="month" axisLine={false} tickLine={false} tick={{ fill: '#687069', fontSize: 12 }} />
                <YAxis axisLine={false} tickLine={false} tickFormatter={chartMoney} tick={{ fill: '#687069', fontSize: 12 }} width={56} />
                <Tooltip formatter={(value) => formatMoney(Number(value))} contentStyle={{ borderRadius: 12, border: '1px solid #dedbd1', boxShadow: '0 10px 30px rgba(20,39,31,.08)' }} />
                <Area type="monotone" dataKey="revenueMinor" stroke="#1f7550" strokeWidth={2.5} fill="url(#revenueFill)" name="Revenue" />
                <Area type="monotone" dataKey="expensesMinor" stroke="#c47a45" strokeWidth={2} fill="transparent" name="Spend" />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </article>

        <article className="panel aging-panel">
          <div className="panel-heading"><div><span className="eyebrow">Collections</span><h2>A/R aging</h2></div></div>
          <div className="aging-total">
            <strong>{formatMoney(metrics.outstandingArMinor, organization.baseCurrency)}</strong>
            <span>total outstanding</span>
          </div>
          <div className="aging-chart">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={aging} layout="vertical" margin={{ left: 0, right: 8 }}>
                <XAxis type="number" hide />
                <YAxis dataKey="bucket" type="category" axisLine={false} tickLine={false} width={76} tick={{ fill: '#687069', fontSize: 12 }} />
                <Tooltip formatter={(value) => formatMoney(Number(value))} cursor={{ fill: '#f4f2eb' }} />
                <Bar dataKey="amountMinor" fill="#d7a16f" radius={[0, 5, 5, 0]} barSize={16} name="Outstanding" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </article>

        <article className="panel budget-panel">
          <div className="panel-heading">
            <div><span className="eyebrow">Controls</span><h2>Budget vs. actual</h2></div>
            <span className="period-chip">This month</span>
          </div>
          <div className="budget-list">
            {budget.map((item) => {
              const used = item.budgetMinor ? Math.min(100, Math.round(item.actualMinor / item.budgetMinor * 100)) : 0
              return (
                <div className="budget-row" key={item.category}>
                  <div className="budget-meta"><strong>{item.category}</strong><span>{formatMoney(item.actualMinor)} of {formatMoney(item.budgetMinor)}</span></div>
                  <div className="progress-track"><span style={{ width: `${used}%` }} className={used > 85 ? 'near-limit' : ''} /></div>
                  <span className="budget-percent">{used}%</span>
                </div>
              )
            })}
          </div>
        </article>
      </section>
    </div>
  )
}
