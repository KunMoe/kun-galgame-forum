// @vitest-environment nuxt
import { describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { mockNuxtImport, mountSuspended } from '@nuxt/test-utils/runtime'
import type { EntityTarget } from '#shared/utils/api/schemas'
import EntityPreviewBody from './EntityPreviewBody.vue'

const { api } = vi.hoisted(() => ({ api: { GET: vi.fn(), POST: vi.fn() } }))

mockNuxtImport('useApiClient', () => () => api)

const author = {
  object: 'user' as const,
  id: '9',
  name: 'shin',
  avatar: null,
  avatar_frame: null
}

const rating: EntityTarget = {
  object: 'rating',
  id: '1',
  work: null,
  author,
  overall: 9,
  short_summary: '好玩',
  has_spoilers: false,
  created_at: '2026-10-01T00:00:00Z'
}

describe('EntityPreviewBody', () => {
  it('draws the rating radar from the rating it loads', async () => {
    api.GET.mockResolvedValue({
      data: {
        aspect_scores: {
          art: 10,
          story: 8,
          music: 8,
          character: 10,
          route: 2,
          system: 10,
          voice: 9,
          replay_value: 6
        }
      },
      response: new Response(null, { status: 200 })
    })
    const wrapper = await mountSuspended(EntityPreviewBody, {
      props: { target: rating }
    })
    await flushPromises()
    expect(api.GET.mock.calls[0]![0]).toBe('/ratings/{rating_id}')
    expect(wrapper.find('svg polygon').exists()).toBe(true)
    expect(wrapper.text()).toContain('画面')
  })
})
