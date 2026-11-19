import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Area, AreaChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import { ArrowDownRight, ArrowUpRight, ChartLine, Plus, SlidersHorizontal, TrendingDown } from 'lucide-react'
import { useEffect, useState, type ReactNode } from 'react'
import { useForm } from 'react-hook-form'
import { formatDate, formatMoney } from '@/shared/lib/format'
import { toMinorUnits } from '@/shared/lib/money'
import { capabilities } from '@/shared/lib/capabilities'
import { queryKeys } from '@/shared/api/queryKeys'
import { Button } from '@/shared/ui/Button'
import { Input, Select } from '@/shared/ui/FormControls'
import { PageHeader } from '@/shared/ui/PageHeader'
import { ErrorState, LoadingState } from '@/shared/ui/ViewState'
import { Modal } from '@/shared/ui/Modal'
import { useAuth } from '@/features/auth/AuthProvider'
import { forecastApi } from './api'
import { adjustmentSchema, scenarioSchema, type AdjustmentForm, type ScenarioForm } from './schema'
import type { CashFlowForecast } from './types'

export function ForecastPage() {
  const { session } = useAuth()
  const queryClient = useQueryClient()
  const [selectedId, setSelectedId] = useState('')
  const [creating, setCreating] = useState(false)
  const [adjusting, setAdjusting] = useState(false)
  const scenarios = useQuery({ queryKey: queryKeys.forecast.scenarios, queryFn: forecastApi.scenarios })
  useEffect(() => { if (!selectedId && scenarios.data?.[0]) setSelectedId(scenarios.data[0].id) }, [scenarios.data, selectedId])
  const forecast = useQuery({ queryKey: queryKeys.forecast.detail(selectedId), queryFn: () => forecastApi.get(selectedId), enabled: Boolean(selectedId) })
  const canEdit = capabilities.operate(session?.user.role)
  const sync = async (value: CashFlowForecast) => {
    queryClient.setQueryData(queryKeys.forecast.detail(value.scenario.id), value)
    await queryClient.invalidateQueries({ queryKey: queryKeys.forecast.scenarios })
  }
  if (scenarios.isLoading) return <LoadingState label="Loading forecast scenarios" />
  if (scenarios.error) return <ErrorState message={scenarios.error.message} />
  const data = forecast.data

  return <div className="page">
    <PageHeader eyebrow="Treasury planning" title="13-week cash-flow forecast" description="Stress-test liquidity with transparent collection, project, expense, and one-off cash assumptions." action={canEdit ? <div className="header-actions"><Button variant="secondary" onClick={() => setCreating(true)}><SlidersHorizontal size={15} /> New scenario</Button><Button disabled={!data} onClick={() => setAdjusting(true)}><Plus size={15} /> Add adjustment</Button></div> : undefined} />
    <section className="forecast-toolbar panel">
      <Select label="Forecast scenario" value={selectedId} onChange={(event) => setSelectedId(event.target.value)}>{scenarios.data?.map((scenario) => <option key={scenario.id} value={scenario.id}>{scenario.name}</option>)}</Select>
      {data ? <div className="forecast-assumptions"><span><strong>{data.scenario.collectionRateBps / 100}%</strong> collections</span><span><strong>{data.scenario.collectionDelayDays} days</strong> delay</span><span><strong>{data.scenario.expenseGrowthBps / 100}%</strong> expense growth</span><span><strong>{data.scenario.milestoneProbabilityBps / 100}%</strong> milestones</span></div> : null}
    </section>
    {forecast.isLoading ? <LoadingState label="Calculating 13-week forecast" /> : forecast.error ? <ErrorState message={forecast.error.message} /> : data ? <>
      <section className="forecast-metrics">
        <div className="panel"><ChartLine /><span>Opening cash</span><strong>{formatMoney(data.openingCashMinor, data.currencyCode)}</strong></div>
        <div className="panel"><ArrowUpRight /><span>Projected inflows</span><strong>{formatMoney(data.totalInflowsMinor, data.currencyCode)}</strong></div>
        <div className="panel"><ArrowDownRight /><span>Projected outflows</span><strong>{formatMoney(data.totalOutflowsMinor, data.currencyCode)}</strong></div>
        <div className={`panel ${data.minimumCashMinor < 0 ? 'forecast-risk' : ''}`}><TrendingDown /><span>Minimum cash · {formatDate(data.minimumCashWeek)}</span><strong>{formatMoney(data.minimumCashMinor, data.currencyCode)}</strong></div>
      </section>
      <section className="panel forecast-chart-panel"><div className="panel-heading"><div><span className="eyebrow">Liquidity runway</span><h2>Weekly ending cash</h2></div></div><div className="forecast-chart"><ResponsiveContainer width="100%" height="100%"><AreaChart data={data.weeks}><defs><linearGradient id="cashForecastFill" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stopColor="#1f7550" stopOpacity={0.25} /><stop offset="100%" stopColor="#1f7550" stopOpacity={0} /></linearGradient></defs><CartesianGrid stroke="#e8e5dc" vertical={false} strokeDasharray="3 3" /><XAxis dataKey="weekStart" tickFormatter={(value) => new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric', timeZone: 'UTC' }).format(new Date(`${value}T00:00:00Z`))} axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: '#687069' }} /><YAxis tickFormatter={(value) => formatMoney(Number(value), data.currencyCode, true)} axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: '#687069' }} width={62} /><Tooltip labelFormatter={(value) => formatDate(String(value))} formatter={(value) => formatMoney(Number(value), data.currencyCode)} /><Area type="monotone" dataKey="endingBalanceMinor" name="Ending cash" stroke="#1f7550" strokeWidth={2.5} fill="url(#cashForecastFill)" /></AreaChart></ResponsiveContainer></div></section>
      <section className="panel table-panel forecast-table"><div className="panel-heading"><div><span className="eyebrow">Driver detail</span><h2>Weekly cash bridge</h2></div></div><div className="table-scroll"><table><thead><tr><th>Week</th><th className="numeric">Opening</th><th className="numeric">Invoice receipts</th><th className="numeric">Project receipts</th><th className="numeric">Manual inflows</th><th className="numeric">Operating outflows</th><th className="numeric">Manual outflows</th><th className="numeric">Ending</th></tr></thead><tbody>{data.weeks.map((week) => <tr key={week.weekStart}><td><strong>{formatDate(week.weekStart)}</strong></td><td className="numeric">{formatMoney(week.openingBalanceMinor, data.currencyCode)}</td><td className="numeric forecast-inflow">{formatMoney(week.invoiceInflowsMinor, data.currencyCode)}</td><td className="numeric forecast-inflow">{formatMoney(week.projectInflowsMinor, data.currencyCode)}</td><td className="numeric forecast-inflow">{formatMoney(week.manualInflowsMinor, data.currencyCode)}</td><td className="numeric">{formatMoney(week.operatingOutflowsMinor, data.currencyCode)}</td><td className="numeric">{formatMoney(week.manualOutflowsMinor, data.currencyCode)}</td><td className={`numeric ${week.endingBalanceMinor < 0 ? 'negative-money' : ''}`}><strong>{formatMoney(week.endingBalanceMinor, data.currencyCode)}</strong></td></tr>)}</tbody></table></div></section>
      {data.adjustments.length ? <section className="panel forecast-adjustments"><div className="panel-heading"><div><span className="eyebrow">Append-only assumptions</span><h2>Manual adjustments</h2></div></div>{data.adjustments.map((item) => <div className="forecast-adjustment" key={item.id}><span className={item.direction === 'inflow' ? 'forecast-inflow' : 'negative-money'}>{item.direction === 'inflow' ? <ArrowUpRight size={15} /> : <ArrowDownRight size={15} />}</span><div><strong>{item.label}</strong><small>{formatDate(item.weekStart)} · {item.createdBy}</small></div><strong>{formatMoney(item.amountMinor, data.currencyCode)}</strong></div>)}</section> : null}
    </> : null}
    <ScenarioDialog open={creating} onClose={() => setCreating(false)} onCreated={(value) => { void sync(value); setSelectedId(value.scenario.id); setCreating(false) }} />
    {data ? <AdjustmentDialog open={adjusting} forecast={data} onClose={() => setAdjusting(false)} onCreated={(value) => { void sync(value); setAdjusting(false) }} /> : null}
  </div>
}

function ScenarioDialog({ open, onClose, onCreated }: { open: boolean; onClose: () => void; onCreated: (value: CashFlowForecast) => void }) {
  const form = useForm<ScenarioForm>({ resolver: zodResolver(scenarioSchema), defaultValues: { name: '', collectionRate: 85, collectionDelayDays: 7, expenseGrowth: 0, milestoneProbability: 70 } })
  const mutation = useMutation({ mutationFn: (value: ScenarioForm) => forecastApi.createScenario({ name: value.name, collectionRateBps: Math.round(value.collectionRate * 100), collectionDelayDays: value.collectionDelayDays, expenseGrowthBps: Math.round(value.expenseGrowth * 100), milestoneProbabilityBps: Math.round(value.milestoneProbability * 100) }), onSuccess: onCreated })
  return <ForecastModal open={open} title="New forecast scenario" onClose={onClose}><form className="forecast-dialog-form" onSubmit={form.handleSubmit((value) => mutation.mutate(value))}><Input label="Scenario name" placeholder="Downside case" error={form.formState.errors.name?.message} {...form.register('name')} /><Input label="Collection rate (%)" type="number" min="0" max="100" step="0.1" error={form.formState.errors.collectionRate?.message} {...form.register('collectionRate', { valueAsNumber: true })} /><Input label="Collection delay (days)" type="number" min="0" max="180" error={form.formState.errors.collectionDelayDays?.message} {...form.register('collectionDelayDays', { valueAsNumber: true })} /><Input label="Expense growth (%)" type="number" min="-100" max="500" step="0.1" error={form.formState.errors.expenseGrowth?.message} {...form.register('expenseGrowth', { valueAsNumber: true })} /><Input label="Milestone probability (%)" type="number" min="0" max="100" step="0.1" error={form.formState.errors.milestoneProbability?.message} {...form.register('milestoneProbability', { valueAsNumber: true })} /><div className="invoice-dialog-actions"><Button type="button" variant="ghost" onClick={onClose}>Cancel</Button><Button type="submit" disabled={mutation.isPending}>Create scenario</Button></div></form></ForecastModal>
}

function AdjustmentDialog({ open, forecast, onClose, onCreated }: { open: boolean; forecast: CashFlowForecast; onClose: () => void; onCreated: (value: CashFlowForecast) => void }) {
  const form = useForm<AdjustmentForm>({ resolver: zodResolver(adjustmentSchema), defaultValues: { weekStart: forecast.weeks[0].weekStart, direction: 'outflow', label: '', amount: 0 } })
  const mutation = useMutation({ mutationFn: (value: AdjustmentForm) => forecastApi.addAdjustment(forecast.scenario.id, { weekStart: value.weekStart, direction: value.direction, label: value.label, amountMinor: toMinorUnits(value.amount) }), onSuccess: onCreated })
  return <ForecastModal open={open} title="Add forecast adjustment" onClose={onClose}><form className="forecast-dialog-form" onSubmit={form.handleSubmit((value) => mutation.mutate(value))}><Select label="Forecast week" error={form.formState.errors.weekStart?.message} {...form.register('weekStart')}>{forecast.weeks.map((week) => <option key={week.weekStart} value={week.weekStart}>{formatDate(week.weekStart)}</option>)}</Select><Select label="Cash direction" error={form.formState.errors.direction?.message} {...form.register('direction')}><option value="inflow">Inflow</option><option value="outflow">Outflow</option></Select><Input label="Description" placeholder="Planned equipment purchase" error={form.formState.errors.label?.message} {...form.register('label')} /><Input label="Amount" type="number" min="0.01" step="0.01" error={form.formState.errors.amount?.message} {...form.register('amount', { valueAsNumber: true })} /><div className="invoice-dialog-actions"><Button type="button" variant="ghost" onClick={onClose}>Cancel</Button><Button type="submit" disabled={mutation.isPending}>Add adjustment</Button></div></form></ForecastModal>
}

function ForecastModal({ open, title, onClose, children }: { open: boolean; title: string; onClose: () => void; children: ReactNode }) {
  return <Modal open={open} title={title} onClose={onClose}>{children}</Modal>
}
