import { afterEach, describe, expect, it, vi } from 'vitest'
import { subscribeToToasts, type ToastEventDetail } from '@/shared/lib/toast'
import { ApiError, apiRequest } from './http'

afterEach(() => {
  vi.unstubAllGlobals()
})

function captureToasts() {
  const messages: ToastEventDetail[] = []
  const unsubscribe = subscribeToToasts((detail) => messages.push(detail))
  return { messages, unsubscribe }
}

describe('apiRequest notifications', () => {
  it('emits a custom success toast for a state-changing response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 201, json: async () => ({ id: '1' }) }))
    const { messages, unsubscribe } = captureToasts()

    await apiRequest('/api/v1/things', { method: 'POST', successMessage: 'Thing created.' })

    expect(messages).toEqual([{ tone: 'success', message: 'Thing created.' }])
    unsubscribe()
  })

  it('emits the server error message for a failed response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 422, json: async () => ({ error: { message: 'Invalid amount.' } }) }))
    const { messages, unsubscribe } = captureToasts()

    await expect(apiRequest('/api/v1/things')).rejects.toEqual(new ApiError('Invalid amount.', 422))

    expect(messages).toEqual([{ tone: 'error', message: 'Invalid amount.' }])
    unsubscribe()
  })

  it('keeps successful read requests quiet', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => ({ items: [] }) }))
    const { messages, unsubscribe } = captureToasts()

    await apiRequest('/api/v1/things')

    expect(messages).toEqual([])
    unsubscribe()
  })
})
