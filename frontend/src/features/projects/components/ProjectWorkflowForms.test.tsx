import { zodResolver } from '@hookform/resolvers/zod'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { useForm } from 'react-hook-form'
import { describe, expect, it, vi } from 'vitest'
import { decisionFormSchema, type DecisionFormValues } from '@/features/projects/schema'
import { DecisionDialog } from './ProjectWorkflowForms'

function DecisionHarness({ onDecision }: { onDecision: (decision: 'approved' | 'rejected', values: DecisionFormValues) => void }) {
  const form = useForm<DecisionFormValues>({ resolver: zodResolver(decisionFormSchema), defaultValues: { note: '' } })
  return <DecisionDialog label="CTR-100" form={form} open pending={false} error={null} onClose={() => undefined} onDecision={onDecision} />
}

describe('project decision workflow', () => {
  it('requires a review note before approving', async () => {
    const onDecision = vi.fn()
    render(<DecisionHarness onDecision={onDecision} />)

    fireEvent.click(screen.getByRole('button', { name: 'Approve' }))
    expect(await screen.findByText('Enter a review note.')).toBeInTheDocument()
    expect(onDecision).not.toHaveBeenCalled()

    fireEvent.change(screen.getByRole('textbox', { name: 'Review note' }), { target: { value: 'Verified independently.' } })
    fireEvent.click(screen.getByRole('button', { name: 'Approve' }))
    await waitFor(() => expect(onDecision).toHaveBeenCalledWith('approved', { note: 'Verified independently.' }))
  })
})
