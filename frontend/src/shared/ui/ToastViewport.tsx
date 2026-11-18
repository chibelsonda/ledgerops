import { AlertCircle, CheckCircle2, X } from 'lucide-react'
import { useEffect, useState } from 'react'
import { subscribeToToasts, type ToastEventDetail } from '../lib/toast'

interface ToastMessage extends ToastEventDetail {
  id: number
}

let nextToastId = 0

function ToastItem({ toast, dismiss }: { toast: ToastMessage; dismiss: () => void }) {
  useEffect(() => {
    const timeout = window.setTimeout(dismiss, 5_000)
    return () => window.clearTimeout(timeout)
  }, [dismiss])

  const Icon = toast.tone === 'success' ? CheckCircle2 : AlertCircle

  return (
    <div className={`toast toast-${toast.tone}`} role={toast.tone === 'error' ? 'alert' : 'status'}>
      <Icon className="toast-icon" size={19} aria-hidden="true" />
      <div className="toast-copy">
        <strong>{toast.tone === 'success' ? 'Success' : 'Something went wrong'}</strong>
        <p>{toast.message}</p>
      </div>
      <button className="toast-dismiss" type="button" aria-label="Dismiss notification" onClick={dismiss}><X size={17} /></button>
    </div>
  )
}

export function ToastViewport() {
  const [toasts, setToasts] = useState<ToastMessage[]>([])

  useEffect(() => subscribeToToasts((detail) => {
    const toast = { ...detail, id: ++nextToastId }
    setToasts((current) => current.some((item) => item.tone === detail.tone && item.message === detail.message)
      ? current
      : [...current.slice(-3), toast])
  }), [])

  function dismiss(id: number) {
    setToasts((current) => current.filter((toast) => toast.id !== id))
  }

  return (
    <div className="toast-viewport" aria-label="Notifications" aria-live="polite" aria-relevant="additions">
      {toasts.map((toast) => <ToastItem key={toast.id} toast={toast} dismiss={() => dismiss(toast.id)} />)}
    </div>
  )
}
