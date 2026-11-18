export type ToastTone = 'success' | 'error'

export interface ToastEventDetail {
  tone: ToastTone
  message: string
}

const toastEventName = 'ledgerops:toast'

export function emitToast(detail: ToastEventDetail) {
  if (typeof window === 'undefined') return
  window.dispatchEvent(new CustomEvent<ToastEventDetail>(toastEventName, { detail }))
}

export function subscribeToToasts(listener: (detail: ToastEventDetail) => void) {
  const handleToast = (event: Event) => listener((event as CustomEvent<ToastEventDetail>).detail)
  window.addEventListener(toastEventName, handleToast)
  return () => window.removeEventListener(toastEventName, handleToast)
}
