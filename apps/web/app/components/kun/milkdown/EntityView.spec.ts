// @vitest-environment nuxt
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { mockNuxtImport, mountSuspended } from '@nuxt/test-utils/runtime'
import EntityCardView from './EntityCardView.vue'
import EntityChipView from './EntityChipView.vue'
import { useKunEditorAdapters } from '~/composables/useKunEditorAdapters'

const { api } = vi.hoisted(() => ({ api: { GET: vi.fn(), POST: vi.fn() } }))

mockNuxtImport('useApiClient', () => () => api)

const topic = (id: string) => ({
  object: 'topic',
  id,
  title: `话题 ${id}`,
  cover_images: [],
  author: {
    object: 'user',
    id: '3',
    name: 'Alice',
    avatar: null,
    avatar_frame: null
  },
  view_count: 1,
  like_count: 0,
  reply_count: 2,
  is_nsfw: false,
  created_at: '2026-10-01T00:00:00Z'
})

const entity = (id: string) => ({
  type: 'topic',
  id,
  url: `https://www.kungal.com/topic/${id}`
})

const ok = (items: unknown[]) => ({
  data: { object: 'list', items },
  response: new Response(null, { status: 200 })
})

describe('editor entity views', () => {
  beforeEach(() => {
    api.GET.mockReset()
  })

  it('batches every preview of one tick into one request', async () => {
    api.GET.mockResolvedValue(ok([topic('101'), topic('102')]))
    const [a, b] = await Promise.all([
      mountSuspended(EntityCardView, {
        props: { entity: entity('101'), selected: false }
      }),
      mountSuspended(EntityChipView, {
        props: { entity: entity('102'), selected: false }
      })
    ])
    await flushPromises()
    expect(api.GET).toHaveBeenCalledTimes(1)
    expect(api.GET.mock.calls[0]![1].params.query).toEqual({
      topic_ids: ['101', '102']
    })
    expect(a.text()).toContain('话题 101')
    expect(b.text()).toBe('话题 102')
    expect(a.find('a').exists()).toBe(false)
    expect(b.find('a').exists()).toBe(false)
  })

  it('reserves the card height while the preview loads', async () => {
    api.GET.mockReturnValue(new Promise(() => {}))
    const view = await mountSuspended(EntityCardView, {
      props: { entity: entity('103'), selected: false }
    })
    expect(view.get('div').classes()).toContain('h-36')
    expect(view.find('.animate-pulse').exists()).toBe(true)
  })

  it('shows the url when the preview cannot be loaded', async () => {
    api.GET.mockResolvedValue({
      error: { status: 503 },
      response: new Response(null, { status: 503 })
    })
    const view = await mountSuspended(EntityChipView, {
      props: { entity: entity('104'), selected: false }
    })
    await flushPromises()
    expect(view.text()).toBe('https://www.kungal.com/topic/104')
  })

  it('never asks the api about a type it does not know', async () => {
    const view = await mountSuspended(EntityCardView, {
      props: {
        entity: { type: 'moyu_patch', id: '1', url: 'https://x.example/1' },
        selected: false
      }
    })
    await flushPromises()
    expect(api.GET).not.toHaveBeenCalled()
    expect(view.text()).toContain('https://x.example/1')
  })
})

describe('useKunEditorAdapters entities', () => {
  it('leaves entity adapters out unless asked', () => {
    expect(useKunEditorAdapters().entityFromUrl).toBeUndefined()
    const adapters = useKunEditorAdapters({ entities: true })
    expect(adapters.entityFromUrl?.('https://kungal.com/topic/9/')).toEqual(
      entity('9')
    )
    expect(adapters.entityTypes?.map((t) => t.type)).toEqual([
      'work',
      'character',
      'company',
      'tag',
      'series',
      'credit_name',
      'rating',
      'topic',
      'galgame_resource'
    ])
    expect(adapters.entityTypes?.[6]).toMatchObject({
      label: 'Galgame 评分卡片',
      command: 'galgame-rate',
      description: '搜索并插入Galgame 评分'
    })
  })

  it('answers an unknown search type with nothing', async () => {
    const adapters = useKunEditorAdapters({ entities: true })
    await expect(
      adapters.searchEntities?.('moyu_patch', 'x', new AbortController().signal)
    ).resolves.toEqual([])
    expect(api.GET).not.toHaveBeenCalled()
  })
})
