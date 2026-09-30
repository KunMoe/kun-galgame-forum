// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockNuxtImport } from '@nuxt/test-utils/runtime'
import { useCloudPreferences } from './useCloudPreferences'

const { api, reportProblem } = vi.hoisted(() => ({
  api: { GET: vi.fn(), PUT: vi.fn() },
  reportProblem: vi.fn()
}))

mockNuxtImport('useApiClient', () => () => api)
mockNuxtImport('reportProblem', () => reportProblem)

const ok = (data: unknown) => ({ data, response: new Response(null) })

const preferences = (doc: Record<string, unknown>, version: number) =>
  ok({ object: 'preferences', doc, version, written_at: null })

const conflict = () => ({
  error: { status: 412, code: 'PRECONDITION_FAILED' },
  response: new Response(null, {
    status: 412,
    headers: { 'content-type': 'application/problem+json' }
  })
})

afterEach(() => {
  vi.useRealTimers()
})

describe('useCloudPreferences', () => {
  it('adopts the newer doc after a 412 without writing it back', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
    api.GET.mockResolvedValueOnce(preferences({}, 0))
    await useCloudPreferences().sync()
    expect(api.PUT).not.toHaveBeenCalled()

    api.PUT.mockResolvedValueOnce(conflict())
    api.GET.mockImplementationOnce(async () =>
      preferences(
        {
          ...api.PUT.mock.calls[0]![1].body.doc,
          show_rating: true,
          rounded: 'lg'
        },
        5
      )
    )
    usePersistGalgameCardStore().showRating = false
    await vi.advanceTimersByTimeAsync(1000)
    expect(api.PUT).toHaveBeenCalledTimes(1)
    expect(usePersistGalgameCardStore().showRating).toBe(true)
    expect(usePersistSettingsStore().showKUNGalgameRounded).toBe('lg')

    await vi.advanceTimersByTimeAsync(5000)
    expect(api.PUT).toHaveBeenCalledTimes(1)
    expect(reportProblem).not.toHaveBeenCalled()
  })
})
