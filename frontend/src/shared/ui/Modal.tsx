import { Dialog, DialogBackdrop, DialogPanel, DialogTitle } from '@headlessui/react'
import { X } from 'lucide-react'
import type { ReactNode } from 'react'
import { Button } from './Button'

interface ModalProps {
  open: boolean
  title: string
  onClose: () => void
  children: ReactNode
}

export function Modal({ open, title, onClose, children }: ModalProps) {
  return (
    <Dialog className="modal" open={open} onClose={onClose}>
      <DialogBackdrop className="modal-backdrop" />
      <div className="modal-positioner">
        <DialogPanel className="modal-panel">
          <div className="close-modal-title">
            <DialogTitle>{title}</DialogTitle>
            <Button aria-label={`Close ${title}`} variant="ghost" onClick={onClose}><X size={18} /></Button>
          </div>
          {children}
        </DialogPanel>
      </div>
    </Dialog>
  )
}
