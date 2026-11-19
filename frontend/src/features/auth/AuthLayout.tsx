import type { ReactNode } from 'react'
import { ShieldCheck } from 'lucide-react'
import { LedgerOpsMark } from '@/shared/ui/LedgerOpsMark'

interface AuthLayoutProps {
  eyebrow: string
  title: string
  description: string
  children: ReactNode
}

export function AuthLayout({ eyebrow, title, description, children }: AuthLayoutProps) {
  return (
    <main className="login-page">
      <section className="login-story" aria-label="LedgerOps introduction">
        <div className="login-brand"><span className="brand-mark"><LedgerOpsMark /></span><span>LedgerOps</span></div>
        <div>
          <span className="eyebrow">Financial command center</span>
          <h1>Decisions start with numbers you can trust.</h1>
          <p>Review cash, receivables, budgets, and exceptions from one focused workspace.</p>
        </div>
        <div className="login-security"><ShieldCheck size={18} /><span>Protected by server-side sessions and role-based access.</span></div>
      </section>
      <section className="login-panel">
        <div className="login-card">
          <span className="eyebrow">{eyebrow}</span>
          <h2>{title}</h2>
          <p>{description}</p>
          {children}
        </div>
      </section>
    </main>
  )
}
