// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { effectScope } from 'vue'
import type { ApiClient } from '#shared/utils/api/client'
import type { Reply, UserRef } from '#shared/utils/api/schemas'
import { useTopicReplies } from './useTopicReplies'

type Get = Pick<ApiClient, 'GET'>['GET']

const user = (): UserRef => ({
  object: 'user',
  id: '1',
  name: 'Alice',
  avatar: null,
  avatar_frame: null
})

const reply = (
  id: string,
  floor: number,
  over: Partial<Reply> = {}
): Reply => ({
  object: 'reply',
  id,
  topic_id: '42',
  floor,
  author: user(),
  author_moemoepoint: 1,
  content: { object: 'document', children: [] },
  like_count: 0,
  dislike_count: 0,
  reactions: [],
  is_pinned: false,
  is_best_answer: false,
  comments: [],
  created_at: '2026-01-01T00:00:00.000Z',
  edited_at: null,
  viewer: null,
  ...over
})

const json = (status: number, body: unknown, contentType: string) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { 'content-type': contentType }
  })

const page = (items: Reply[], next?: string) => ({
  object: 'list' as const,
  items,
  ...(next ? { next_cursor: next } : {})
})

const ok = (data: unknown, status = 200) => ({
  data,
  response: json(status, data, 'application/json')
})

const problemBody = (status: number, code: string) => ({
  type: 'about:blank',
  title: 'Error',
  status,
  code,
  request_id: 'req_01ARZ3NDEKTSV4RRFFQ69G5FAV',
  errors: []
})

const fail = (status: number, code: string) => ({
  error: problemBody(status, code),
  response: json(status, problemBody(status, code), 'application/problem+json')
})

type Call = {
  path: string
  query?: Record<string, unknown>
  replyId?: string
}

const run = (topicId: string, get: Get) => {
  const scope = effectScope()
  const box = scope.run(() => useTopicReplies(topicId, { GET: get }))
  return { box: box!, stop: () => scope.stop() }
}

const captureGet = (
  handler: (call: Call) => ReturnType<typeof ok> | ReturnType<typeof fail>
) => {
  const calls: Call[] = []
  const get = vi.fn(
    async (
      path: string,
      init?: {
        params?: {
          query?: Record<string, unknown>
          path?: { reply_id?: string }
        }
      }
    ) => {
      const call: Call = {
        path,
        query: init?.params?.query,
        replyId: init?.params?.path?.reply_id
      }
      calls.push(call)
      return handler(call)
    }
  )
  return { get: get as unknown as Get, calls }
}

afterEach(() => {
  clearNuxtData()
})

describe('useTopicReplies', () => {
  it('loads the first page with limit 30 and sort=floor_asc', async () => {
    const { get, calls } = captureGet(() =>
      ok(page([reply('1', 1), reply('2', 2)], 'c1'))
    )
    const { box, stop } = run('t-initial', get)
    await box.loadInitialReplies()
    expect(calls[0]).toMatchObject({
      path: '/topics/{topic_id}/replies',
      query: { limit: 30, sort: 'floor_asc' }
    })
    expect(calls[0]!.query).not.toHaveProperty('from_floor')
    expect(box.replies.value.map((row) => row.id)).toEqual(['1', '2'])
    expect(box.isComplete.value).toBe(false)
    expect(box.hasEarlier.value).toBe(false)
    stop()
  })

  it('appends loadMore in order and skips duplicate ids', async () => {
    const { get } = captureGet((call) => {
      if (!call.query?.cursor) {
        return ok(page([reply('1', 1), reply('2', 2)], 'c2'))
      }
      expect(call.query.cursor).toBe('c2')
      expect(call.query).not.toHaveProperty('from_floor')
      return ok(page([reply('2', 2), reply('3', 3)]))
    })
    const { box, stop } = run('t-forward', get)
    await box.loadInitialReplies()
    await box.loadMore()
    expect(box.replies.value.map((row) => row.id)).toEqual(['1', '2', '3'])
    expect(box.replies.value.map((row) => row.floor)).toEqual([1, 2, 3])
    expect(box.isComplete.value).toBe(true)
    stop()
  })

  it('prepends earlier pages in ascending floor order and clears hasEarlier when there is no next_cursor', async () => {
    let earlier = 0
    const { get, calls } = captureGet((call) => {
      if (call.query?.sort === 'floor_desc') {
        earlier++
        if (earlier === 1) {
          expect(call.query.from_floor).toBe(4)
          expect(call.query.cursor).toBeUndefined()
          return ok(page([reply('4', 4), reply('3', 3), reply('2', 2)], 'b1'))
        }
        expect(call.query.cursor).toBe('b1')
        expect(call.query.from_floor).toBe(4)
        return ok(page([reply('1', 1)]))
      }
      return ok(page([reply('5', 5), reply('6', 6)], 'f1'))
    })
    const { box, stop } = run('t-back', get)
    await box.loadInitialReplies({ fromFloor: 5 })
    expect(box.hasEarlier.value).toBe(true)
    expect(calls[0]!.query).toMatchObject({
      sort: 'floor_asc',
      from_floor: 5,
      limit: 30
    })
    await box.loadEarlier()
    expect(box.replies.value.map((row) => row.floor)).toEqual([2, 3, 4, 5, 6])
    expect(box.hasEarlier.value).toBe(true)
    await box.loadEarlier()
    expect(box.replies.value.map((row) => row.floor)).toEqual([
      1, 2, 3, 4, 5, 6
    ])
    expect(box.hasEarlier.value).toBe(false)
    stop()
  })

  it('reloads from the first floor when sort switches', async () => {
    const { get, calls } = captureGet((call) => {
      if (call.query?.sort === 'floor_desc') {
        return ok(page([reply('9', 9), reply('8', 8)]))
      }
      return ok(page([reply('1', 1)], 'c1'))
    })
    const { box, stop } = run('t-sort', get)
    await box.loadInitialReplies()
    await box.setSort('desc')
    expect(calls[1]!.query).toMatchObject({
      sort: 'floor_desc',
      limit: 30
    })
    expect(calls[1]!.query).not.toHaveProperty('from_floor')
    expect(box.replies.value.map((row) => row.id)).toEqual(['9', '8'])
    expect(box.hasEarlier.value).toBe(false)
    expect(box.isComplete.value).toBe(true)
    expect(box.sortOrder.value).toBe('desc')
    stop()
  })

  it('sends from_floor when loadInitialReplies is given an anchor', async () => {
    const { get, calls } = captureGet(() => ok(page([reply('7', 7)])))
    const { box, stop } = run('t-anchor', get)
    await box.loadInitialReplies({ fromFloor: 7 })
    expect(calls[0]!.query?.from_floor).toBe(7)
    expect(box.hasEarlier.value).toBe(true)
    stop()
  })

  it('sends the anchor again with every forward cursor it issued', async () => {
    const { get, calls } = captureGet((call) => {
      if (call.query?.cursor === 'f1') {
        return ok(page([reply('33', 33)], 'f2'))
      }
      if (call.query?.cursor === 'f2') {
        return ok(page([reply('34', 34)]))
      }
      return ok(page([reply('31', 31), reply('32', 32)], 'f1'))
    })
    const { box, stop } = run('t-anchor-forward', get)
    await box.loadInitialReplies({ fromFloor: 31 })
    await box.loadMore()
    await box.loadMore()
    expect(calls.map((call) => call.query?.from_floor)).toEqual([31, 31, 31])
    expect(calls.map((call) => call.query?.cursor)).toEqual([
      undefined,
      'f1',
      'f2'
    ])
    expect(box.replies.value.map((row) => row.floor)).toEqual([31, 32, 33, 34])
    expect(box.problem.value).toBeNull()
    stop()
  })

  it('drops the anchor once the sort switches', async () => {
    const { get, calls } = captureGet((call) => {
      if (call.query?.sort === 'floor_desc') {
        return call.query.cursor
          ? ok(page([reply('7', 7)]))
          : ok(page([reply('9', 9), reply('8', 8)], 'd1'))
      }
      return ok(page([reply('31', 31)], 'f1'))
    })
    const { box, stop } = run('t-anchor-sort', get)
    await box.loadInitialReplies({ fromFloor: 31 })
    await box.setSort('desc')
    await box.loadMore()
    expect(calls[2]!.query).toMatchObject({ sort: 'floor_desc', cursor: 'd1' })
    expect(calls[2]!.query).not.toHaveProperty('from_floor')
    stop()
  })

  it('reads back from the anchor when the anchored page is empty', async () => {
    const { get, calls } = captureGet((call) => {
      if (call.query?.sort === 'floor_desc') {
        return ok(page([reply('63', 63), reply('62', 62)]))
      }
      return ok(page([]))
    })
    const { box, stop } = run('t-anchor-empty', get)
    await box.loadInitialReplies({ fromFloor: 91 })
    await box.loadEarlier()
    expect(calls[1]!.query).toMatchObject({
      sort: 'floor_desc',
      from_floor: 90
    })
    expect(box.replies.value.map((row) => row.floor)).toEqual([62, 63])
    stop()
  })

  it('does not duplicate an id already loaded from the other direction', async () => {
    const { get } = captureGet((call) => {
      if (call.query?.sort === 'floor_desc') {
        return ok(page([reply('5', 5), reply('4', 4)]))
      }
      return ok(page([reply('5', 5), reply('6', 6)], 'f1'))
    })
    const { box, stop } = run('t-dedupe', get)
    await box.loadInitialReplies({ fromFloor: 5 })
    await box.loadEarlier()
    expect(box.replies.value.map((row) => row.id)).toEqual(['4', '5', '6'])
    stop()
  })

  it('updates a loaded reply in place on refresh', async () => {
    const { get } = captureGet((call) => {
      if (call.path === '/replies/{reply_id}') {
        return ok(reply('1', 1, { like_count: 9 }))
      }
      return ok(page([reply('1', 1, { like_count: 0 })]))
    })
    const { box, stop } = run('t-refresh-update', get)
    await box.loadInitialReplies()
    await box.refreshReply('1')
    expect(box.replies.value).toHaveLength(1)
    expect(box.replies.value[0]!.like_count).toBe(9)
    stop()
  })

  it('appends a reply that is not loaded yet on refresh', async () => {
    const { get } = captureGet((call) => {
      if (call.path === '/replies/{reply_id}') {
        expect(call.replyId).toBe('99')
        return ok(reply('99', 99))
      }
      return ok(page([reply('1', 1)]))
    })
    const { box, stop } = run('t-refresh-append', get)
    await box.loadInitialReplies()
    await box.refreshReply('99')
    expect(box.replies.value.map((row) => row.id)).toEqual(['1', '99'])
    stop()
  })

  it('puts a refreshed new reply first when the list reads from the last floor', async () => {
    const { get } = captureGet((call) => {
      if (call.path === '/replies/{reply_id}') {
        return ok(reply('10', 10))
      }
      if (call.query?.sort === 'floor_desc') {
        return ok(page([reply('9', 9), reply('8', 8)]))
      }
      return ok(page([reply('1', 1)]))
    })
    const { box, stop } = run('t-refresh-desc', get)
    await box.loadInitialReplies()
    await box.setSort('desc')
    await box.refreshReply('10')
    expect(box.replies.value.map((row) => row.id)).toEqual(['10', '9', '8'])
    stop()
  })

  it('removes a loaded reply when refresh returns 404', async () => {
    const { get } = captureGet((call) => {
      if (call.path === '/replies/{reply_id}') {
        return fail(404, 'NOT_FOUND')
      }
      return ok(page([reply('1', 1), reply('2', 2)]))
    })
    const { box, stop } = run('t-refresh-404', get)
    await box.loadInitialReplies()
    await box.refreshReply('1')
    expect(box.replies.value.map((row) => row.id)).toEqual(['2'])
    expect(box.problem.value).toBeNull()
    stop()
  })

  it('keeps a problem and retries the failed load', async () => {
    let initial = 0
    const { get } = captureGet(() => {
      initial++
      if (initial === 1) {
        return fail(400, 'INVALID_PARAMETER')
      }
      return ok(page([reply('1', 1)]))
    })
    const { box, stop } = run('t-problem', get)
    await box.loadInitialReplies()
    expect(box.problem.value).toMatchObject({
      status: 400,
      code: 'INVALID_PARAMETER'
    })
    expect(box.replies.value).toEqual([])
    await box.retry()
    expect(box.problem.value).toBeNull()
    expect(box.replies.value.map((row) => row.id)).toEqual(['1'])
    stop()
  })
})
