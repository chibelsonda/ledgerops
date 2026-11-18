import { AlertTriangle, LoaderCircle } from 'lucide-react'

export function LoadingState({ label = 'Loading financial data' }: { label?: string }) {
  return <div className="view-state"><LoaderCircle className="spin" /><span>{label}…</span></div>
}

export function ErrorState({ message }: { message: string }) {
  return <div className="view-state error-state"><AlertTriangle /><div><strong>Couldn’t load this view</strong><span>{message}</span></div></div>
}

