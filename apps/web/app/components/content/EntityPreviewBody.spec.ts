// @vitest-environment nuxt
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, type VueWrapper } from '@vue/test-utils'
import { mockNuxtImport, mountSuspended } from '@nuxt/test-utils/runtime'
import type { EntityTarget } from '#shared/utils/api/schemas'
import EntityPreviewBody from './EntityPreviewBody.vue'

const { api } = vi.hoisted(() => ({ api: { GET: vi.fn(), POST: vi.fn() } }))

mockNuxtImport('useApiClient', () => () => api)

let wrapper: VueWrapper | undefined

beforeEach(() => vi.clearAllMocks())
afterEach(() => wrapper?.unmount())

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
    wrapper = await mountSuspended(EntityPreviewBody, {
      props: { target: rating }
    })
    await flushPromises()
    expect(api.GET.mock.calls[0]![0]).toBe('/ratings/{rating_id}')
    expect(wrapper.find('svg polygon').exists()).toBe(true)
    expect(wrapper.text()).toContain('画面')
  })

  it.each(['portion', 'serious'])(
    'loads the complete %s review and reveals its inline spoiler after confirmation',
    async (level) => {
      const summary = `${'普通评价'.repeat(60)}||结局真相||完整短评的结尾`
      api.GET.mockResolvedValue({
        data: {
          short_summary: summary,
          spoiler_level: level,
          aspect_scores: {}
        },
        response: new Response(null, { status: 200 })
      })
      wrapper = await mountSuspended(EntityPreviewBody, {
        props: {
          target: {
            ...rating,
            has_spoilers: true,
            short_summary: '普通评价███'
          }
        }
      })
      await flushPromises()

      expect(wrapper.text()).not.toContain('完整短评的结尾')
      const confirm = wrapper
        .findAll('button')
        .find((button) => button.text().includes('这篇评分含有剧透，点击查看'))
      expect(confirm).toBeDefined()
      await confirm!.trigger('click')

      expect(wrapper.text()).toContain('完整短评的结尾')
      expect(wrapper.text()).not.toContain('███')
      const spoiler = wrapper.get('.kun-spoiler-hidden')
      expect(spoiler.text()).toBe('结局真相')
      await spoiler.trigger('click')
      expect(wrapper.find('.kun-spoiler-hidden').exists()).toBe(false)
    }
  )

  it('shows the full unflagged review with clickable inline spoilers', async () => {
    api.GET.mockResolvedValue({
      data: {
        short_summary: '前文||结局真相||完整短评的结尾',
        spoiler_level: 'none',
        aspect_scores: {}
      },
      response: new Response(null, { status: 200 })
    })
    wrapper = await mountSuspended(EntityPreviewBody, {
      props: { target: { ...rating, short_summary: '前文███' } }
    })
    await flushPromises()

    expect(wrapper.text()).toContain('完整短评的结尾')
    expect(wrapper.text()).not.toContain('这篇评分含有剧透，点击查看')
    expect(wrapper.get('.kun-spoiler-hidden').text()).toBe('结局真相')
  })

  it('respects the current spoiler level when the cached preview is unflagged', async () => {
    api.GET.mockResolvedValue({
      data: {
        short_summary: '最新的结局真相',
        spoiler_level: 'serious',
        aspect_scores: {}
      },
      response: new Response(null, { status: 200 })
    })
    wrapper = await mountSuspended(EntityPreviewBody, {
      props: { target: rating }
    })
    await flushPromises()

    expect(wrapper.text()).not.toContain('最新的结局真相')
    expect(wrapper.text()).toContain('这篇评分含有剧透，点击查看')
  })
})
