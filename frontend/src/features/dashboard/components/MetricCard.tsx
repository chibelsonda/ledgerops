import type { LucideIcon } from 'lucide-react'

interface MetricCardProps {
  label: string
  value: string
  detail: string
  icon: LucideIcon
  tone?: 'default' | 'warning'
}

export function MetricCard({ label, value, detail, icon: Icon, tone = 'default' }: MetricCardProps) {
  return (
    <article className={`metric-card ${tone === 'warning' ? 'metric-warning' : ''}`}>
      <div className="metric-top"><span>{label}</span><Icon size={19} strokeWidth={1.7} /></div>
      <strong>{value}</strong><small>{detail}</small>
    </article>
  )
}

