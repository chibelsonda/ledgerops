import { titleCase } from '@/shared/lib/format'

export function StatusBadge({ status }: { status: string }) {
  return <span className={`status-badge status-${status}`}>{titleCase(status)}</span>
}

