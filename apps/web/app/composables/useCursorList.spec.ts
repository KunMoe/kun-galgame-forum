// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { effectScope, ref, type MaybeRefOrGetter } from 'vue'
import { mockNuxtImport } from '@nuxt/test-utils/runtime'
import { useCursorList } from './useCursorList'

const { isHistoryPop, reportProblem } = vi.hoisted(() => ({
  isHistoryPop: vi.fn(() => false),
  reportProblem: vi.fn()
}))

vi.mock('~/utils/historyPop', () => ({
  isHistoryPop,
  trackHistoryPops: vi.fn()
}))
mockNuxtImport('reportProblem', () => reportProblem)

type Item = { id: string; title: string }

const item = (id: string, title = id): Item => ({ id, title })

const ok = (items: Item[], next?: string) => ({
  data: {
    items,
    ...(next ? { next_cursor: next } : {})
  },
  response: new Response(null, { status: 200 })
})

const problemBody = {
  type: 'about:blank',
  title: 'Invalid parameter',
  status: 400,
  code: 'INVALID_PARAMETER',
  request_id: 'req_01ARZ3NDEKTSV4RRFFQ69G5FAV',
  errors: []
}

const fail = () => ({
  error: problemBody,
  response: new Response(JSON.stringify(problemBody), {
    status: 400,
    headers: { 'content-type': 'application/problem+json' }
  })
})

const runList = async (
  key: MaybeRefOrGetter<string>,
  fetchPage: Parameters<typeof useCursorList<Item>>[1]
) => {
  const scope = effectScope()
  const list = await scope.run(() => useCursorList<Item>(key, fetchPage))
  return { list: list!, stop: () => scope.stop() }
}

afterEach(() => {
  isHistoryPop.mockReturnValue(false)
  reportProblem.mockReset()
  const nuxtApp = useNuxtApp()
  nuxtApp.isHydrating = false
  clearNuxtData()
})

describe('useCursorList', () => {
  it('uses payload data while hydrating and does not request again', async () => {
    const fetchPage = vi.fn(async () => ok([item('h1')], 'n'))
    const nuxtApp = useNuxtApp()
    nuxtApp.payload.data['cursor:hydrate-1'] = {
      ok: true,
      data: { items: [item('h1')], next_cursor: 'n' }
    }
    nuxtApp.isHydrating = true
    const { list, stop } = await runList('hydrate-1', fetchPage)
    expect(fetchPage).not.toHaveBeenCalled()
    expect(list.items.value).toEqual([item('h1')])
    expect(list.hasMore.value).toBe(true)
    stop()
  })

  it('exposes the total a page carries and keeps it when a later page has none', async () => {
    const withTotal = (items: Item[], total: number, next?: string) => {
      const page = ok(items, next)
      return { ...page, data: { ...page.data, total } }
    }
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce(withTotal([item('1')], 3, 'c2'))
      .mockResolvedValueOnce(ok([item('2')], 'c3'))
      .mockResolvedValueOnce(withTotal([item('3')], 2))
    const { list, stop } = await runList('total-1', fetchPage)
    expect(list.total.value).toBe(3)
    await list.loadMore()
    expect(list.total.value).toBe(3)
    await list.loadMore()
    expect(list.total.value).toBe(2)
    stop()
  })

  it('has no total when the collection sends none', async () => {
    const { list, stop } = await runList('total-2', async () => ok([item('1')]))
    expect(list.total.value).toBeUndefined()
    stop()
  })

  it('loads the first page, appends loadMore, and skips duplicate ids', async () => {
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce(ok([item('1'), item('2')], 'c2'))
      .mockResolvedValueOnce(ok([item('2'), item('3')]))
    const { list, stop } = await runList('append-1', fetchPage)
    expect(list.items.value.map((row) => row.id)).toEqual(['1', '2'])
    expect(list.hasMore.value).toBe(true)
    await list.loadMore()
    expect(fetchPage).toHaveBeenCalledTimes(2)
    expect(fetchPage.mock.calls[1]![1]).toBe('c2')
    expect(list.items.value.map((row) => row.id)).toEqual(['1', '2', '3'])
    expect(list.hasMore.value).toBe(false)
    stop()
  })

  it('is a no-op while loadMore is in flight or hasMore is false', async () => {
    let resolveMore: ((value: ReturnType<typeof ok>) => void) | undefined
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce(ok([item('1')], 'c2'))
      .mockImplementationOnce(
        () =>
          new Promise<ReturnType<typeof ok>>((resolve) => {
            resolveMore = resolve
          })
      )
    const { list, stop } = await runList('inflight-1', fetchPage)
    const first = list.loadMore()
    const second = list.loadMore()
    expect(fetchPage).toHaveBeenCalledTimes(2)
    resolveMore!(ok([item('2')]))
    await first
    await second
    expect(list.items.value.map((row) => row.id)).toEqual(['1', '2'])
    await list.loadMore()
    expect(fetchPage).toHaveBeenCalledTimes(2)
    stop()
  })

  it('reports a failed loadMore and leaves hasMore true', async () => {
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce(ok([item('1')], 'c2'))
      .mockResolvedValueOnce(fail())
    const { list, stop } = await runList('fail-more-1', fetchPage)
    await list.loadMore()
    expect(reportProblem).toHaveBeenCalledTimes(1)
    expect(list.hasMore.value).toBe(true)
    expect(list.items.value.map((row) => row.id)).toEqual(['1'])
    stop()
  })

  it('loads a new key first page and drops old items', async () => {
    const key = ref('page-a')
    const fetchPage = vi.fn(async () => {
      if (key.value === 'page-a') {
        return ok([item('a')], 'ca')
      }
      return ok([item('b')])
    })
    const { list, stop } = await runList(key, fetchPage)
    expect(list.items.value.map((row) => row.id)).toEqual(['a'])
    key.value = 'page-b'
    await vi.waitFor(() => {
      expect(list.items.value.map((row) => row.id)).toEqual(['b'])
    })
    expect(fetchPage).toHaveBeenCalledTimes(2)
    expect(list.hasMore.value).toBe(false)
    stop()
  })

  it('discards a stale loadMore when the key has changed', async () => {
    const key = ref('stale-a')
    let resolveMore: ((value: ReturnType<typeof ok>) => void) | undefined
    const fetchPage = vi.fn(async () => {
      if (key.value === 'stale-a') {
        return ok([item('a1')], 'ca')
      }
      return ok([item('b1')])
    })
    const { list, stop } = await runList(key, fetchPage)
    fetchPage.mockImplementationOnce(
      () =>
        new Promise<ReturnType<typeof ok>>((resolve) => {
          resolveMore = resolve
        })
    )
    const pending = list.loadMore()
    key.value = 'stale-b'
    await vi.waitFor(() => {
      expect(list.items.value.map((row) => row.id)).toEqual(['b1'])
    })
    resolveMore!(ok([item('a2')]))
    await pending
    expect(list.items.value.map((row) => row.id)).toEqual(['b1'])
    stop()
  })

  it('does not put loadMore pages into the Nuxt payload', async () => {
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce(ok([item('1')], 'c2'))
      .mockResolvedValueOnce(ok([item('2')]))
    const { list, stop } = await runList('payload-1', fetchPage)
    await list.loadMore()
    const stored = useNuxtApp().payload.data['cursor:payload-1'] as {
      ok: boolean
      data: { items: Item[] }
    }
    expect(stored.data.items.map((row) => row.id)).toEqual(['1'])
    expect(list.items.value.map((row) => row.id)).toEqual(['1', '2'])
    stop()
  })

  it('restores a snapshot on a history pop without calling fetchPage', async () => {
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce(ok([item('1')], 'c2'))
      .mockResolvedValueOnce(ok([item('2')]))
    const first = await runList('pop-1', fetchPage)
    await first.list.loadMore()
    expect(fetchPage).toHaveBeenCalledTimes(2)
    first.stop()

    isHistoryPop.mockReturnValue(true)
    const second = await runList('pop-1', fetchPage)
    expect(fetchPage).toHaveBeenCalledTimes(2)
    expect(second.list.items.value.map((row) => row.id)).toEqual(['1', '2'])
    expect(second.list.hasMore.value).toBe(false)
    second.stop()
  })

  it('refetches on refresh during a pop instead of serving the snapshot', async () => {
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce(ok([item('1')], 'c2'))
      .mockResolvedValueOnce(ok([item('2')]))
      .mockResolvedValueOnce(ok([item('7')], 'again'))
    const first = await runList('pop-refresh', fetchPage)
    await first.list.loadMore()
    first.stop()

    isHistoryPop.mockReturnValue(true)
    const second = await runList('pop-refresh', fetchPage)
    expect(fetchPage).toHaveBeenCalledTimes(2)
    await second.list.refresh()
    expect(fetchPage).toHaveBeenCalledTimes(3)
    expect(second.list.items.value.map((row) => row.id)).toEqual(['7'])
    second.stop()
  })

  it('fetches the first page on a push even when a snapshot exists', async () => {
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce(ok([item('1')], 'c2'))
      .mockResolvedValueOnce(ok([item('2')]))
      .mockResolvedValueOnce(ok([item('9')], 'fresh'))
    const first = await runList('push-1', fetchPage)
    await first.list.loadMore()
    first.stop()

    isHistoryPop.mockReturnValue(false)
    const second = await runList('push-1', fetchPage)
    expect(fetchPage).toHaveBeenCalledTimes(3)
    expect(second.list.items.value.map((row) => row.id)).toEqual(['9'])
    expect(second.list.hasMore.value).toBe(true)
    second.stop()
  })

  it('drops the oldest snapshot after 10 keys', async () => {
    const fetchPage = vi.fn(async (_api, _cursor, _ctx) => {
      const id = String(_cursor ?? 'first')
      return ok([item(id)])
    })
    for (let i = 0; i < 11; i++) {
      fetchPage.mockResolvedValueOnce(ok([item(String(i))], `n${i}`))
      const run = await runList(`lru-${i}`, fetchPage)
      run.stop()
    }
    isHistoryPop.mockReturnValue(true)
    fetchPage.mockClear()
    fetchPage.mockResolvedValue(ok([item('fetched')]))

    const dropped = await runList('lru-0', fetchPage)
    expect(fetchPage).toHaveBeenCalled()
    dropped.stop()

    fetchPage.mockClear()
    const kept = await runList('lru-10', fetchPage)
    expect(fetchPage).not.toHaveBeenCalled()
    expect(kept.list.items.value.map((row) => row.id)).toEqual(['10'])
    kept.stop()
  })
})
