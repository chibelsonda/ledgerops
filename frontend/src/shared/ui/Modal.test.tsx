import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { Modal } from './Modal'

describe('Modal', () => {
  it('provides a named close action', () => {
    const onClose = vi.fn()
    render(<Modal open title="Review contract" onClose={onClose}><p>Review details</p></Modal>)

    expect(screen.getByRole('heading', { name: 'Review contract' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Close Review contract' }))
    expect(onClose).toHaveBeenCalledOnce()
  })
})
