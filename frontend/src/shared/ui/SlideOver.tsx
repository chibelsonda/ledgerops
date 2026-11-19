import { Dialog, DialogBackdrop, DialogPanel, DialogTitle } from '@headlessui/react'
import { X } from 'lucide-react'
import type { ReactNode } from 'react'
import { Button } from './Button'

interface SlideOverProps {
  open: boolean
  eyebrow: string
  title: ReactNode
  icon: ReactNode
  closeLabel: string
  panelClassName?: string
  onClose: () => void
  children: ReactNode
}

export function SlideOver({ open, eyebrow, title, icon, closeLabel, panelClassName = '', onClose, children }: SlideOverProps) {
  return (
    <Dialog className="slide-over" open={open} onClose={onClose}>
      <DialogBackdrop transition className="slide-over-backdrop" />
      <div className="slide-over-positioner">
        <DialogPanel transition className={`slide-over-panel ${panelClassName}`.trim()}>
          <div className="slide-over-heading">
            <div className="slide-over-title-icon">{icon}</div>
            <div><span className="eyebrow">{eyebrow}</span><DialogTitle>{title}</DialogTitle></div>
            <Button aria-label={closeLabel} className="slide-over-close" variant="ghost" onClick={onClose}><X size={24} /></Button>
          </div>
          {children}
        </DialogPanel>
      </div>
    </Dialog>
  )
}
