import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { ClientDialog } from './ClientDialog'

describe('ClientDialog', () => {
  it('keeps create validation inline and does not call the API', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch')
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
    render(
      <QueryClientProvider client={queryClient}>
        <ClientDialog open client={null} onClose={() => undefined} />
      </QueryClientProvider>,
    )

    fireEvent.click(screen.getByRole('button', { name: 'Create client' }))

    expect(await screen.findByText('Enter the client name.')).toBeInTheDocument()
    expect(screen.getByText('Enter the contact name.')).toBeInTheDocument()
    expect(screen.getByText('Enter a valid email address.')).toBeInTheDocument()
    expect(fetchMock).not.toHaveBeenCalled()
    fetchMock.mockRestore()
  })
})
