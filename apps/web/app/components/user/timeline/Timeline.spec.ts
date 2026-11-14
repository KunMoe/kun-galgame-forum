// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import Timeline from './Timeline.vue'

const item = (id: string) => ({
  object: 'following_activity_item',
  id,
  site: 'kungal',
  key: `k${id}`,
  verb: 'publish',
  object_kind: 'topic',
  object_label: '话题',
  title: `Topic ${id}`,
  excerpt: '',
  url: `https://www.kungal.com/topic/${id}`,
  in_site_path: `/topic/${id}`,
  cover: null,
  related_work_id: null,
  is_nsfw: false,
  occurred_at: '2026-11-14T08:00:00Z',
  topic_times: null
})

const group = (id: string, date: string, verb = 'publish', count = 1) => ({
  object: 'user_activity_group',
  id,
  site: 'kungal',
  verb,
  object_kind: 'topic',
  object_label: '话题',
  calendar_date: date,
  item_count: count,
  latest_at: `${date}T08:00:00Z`,
  items: [item(`${id}1`)]
})

const page = (items: unknown[], next?: string) =>
  new Response(
    JSON.stringify({
      object: 'list',
      items,
      ...(next ? { next_cursor: next } : {})
    }),
    { status: 200, headers: { 'content-type': 'application/json' } }
  )

const requests: URL[] = []

const serve = (respond: (url: URL) => Response) =>
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: Request | string) => {
      const url = new URL(typeof input === 'string' ? input : input.url)
      requests.push(url)
      return respond(url)
    })
  )

afterEach(() => {
  requests.length = 0
  vi.unstubAllGlobals()
})

describe('UserTimeline', () => {
  it('splits groups into days and appends the next page on demand', async () => {
    serve((url) =>
      url.searchParams.get('cursor') === 'cur_2'
        ? page([group('3', '2026-11-12', 'rate')])
        : page(
            [
              group('1', '2026-11-14', 'publish', 3),
              group('2', '2026-11-14', 'reply')
            ],
            'cur_2'
          )
    )
    const wrapper = await mountSuspended(Timeline, { props: { userId: 2 } })
    await flushPromises()

    expect(requests[0]?.pathname.endsWith('/users/2/activities')).toBe(true)
    expect(Object.fromEntries(requests[0]!.searchParams)).toEqual({
      include_nsfw: 'false',
      limit: '20'
    })
    expect(wrapper.findAll('section h3').map((h) => h.text())).toEqual([
      '11 月 14 日 周六'
    ])
    expect(wrapper.text()).toContain('发布了 3 个话题')

    const more = wrapper.findAll('button').find((b) => b.text() === '加载更多')!
    await more.trigger('click')
    await flushPromises()

    expect(wrapper.findAll('section h3').map((h) => h.text())).toEqual([
      '11 月 14 日 周六',
      '11 月 12 日 周四'
    ])
    expect(wrapper.findAll('button').some((b) => b.text() === '加载更多')).toBe(
      false
    )
  })

  it('says so when there is nothing to show', async () => {
    serve(() => page([]))
    const wrapper = await mountSuspended(Timeline, { props: { userId: 3 } })
    await flushPromises()
    expect(wrapper.text()).toContain('还没有动态')
  })
})
