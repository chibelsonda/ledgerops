import { afterEach, describe, expect, it, vi } from 'vitest'
import { subscribeToToasts, type ToastEventDetail } from '@/shared/lib/toast'
import { ApiError, apiFormRequest, apiRequest } from './http'

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
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 201, json: async () => ({ id: '1' }) })
    vi.stubGlobal('fetch', fetchMock)
    const { messages, unsubscribe } = captureToasts()

    await apiRequest('/things', { method: 'POST', successMessage: 'Thing created.' })

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/things', expect.any(Object))
    expect(messages).toEqual([{ tone: 'success', message: 'Thing created.' }])
    unsubscribe()
  })

  it('emits the server error message for a failed response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 422, json: async () => ({ error: { message: 'Invalid amount.' } }) }))
    const { messages, unsubscribe } = captureToasts()

    await expect(apiRequest('/things')).rejects.toEqual(new ApiError('Invalid amount.', 422))

    expect(messages).toEqual([{ tone: 'error', message: 'Invalid amount.' }])
    unsubscribe()
  })

  it('keeps a form-handled error inline without suppressing future success messages', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce({ ok: false, status: 422, json: async () => ({ error: { message: 'Invalid amount.' } }) })
      .mockResolvedValueOnce({ ok: true, status: 200, json: async () => ({ id: '1' }) })
    vi.stubGlobal('fetch', fetchMock)
    const { messages, unsubscribe } = captureToasts()

    await expect(apiFormRequest('/things', { method: 'POST' }))
      .rejects.toEqual(new ApiError('Invalid amount.', 422))
    await apiFormRequest('/things', { method: 'POST', successMessage: 'Thing saved.' })

    expect(messages).toEqual([{ tone: 'success', message: 'Thing saved.' }])
    unsubscribe()
  })

  it('keeps successful read requests quiet', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => ({ items: [] }) }))
    const { messages, unsubscribe } = captureToasts()

    await apiRequest('/things')

    expect(messages).toEqual([])
    unsubscribe()
  })
})
