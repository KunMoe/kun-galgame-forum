// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockNuxtImport } from '@nuxt/test-utils/runtime'
import type { Topic, TopicSource, UserRef } from '#shared/utils/api/schemas'
import { applyTopicSource } from './applyTopicSource'
import { useTopicSubmitter } from './useTopicSubmitter'

const { navigateTo, reportProblem, api } = vi.hoisted(() => ({
  navigateTo: vi.fn(),
  reportProblem: vi.fn(),
  api: {
    POST: vi.fn(),
    PATCH: vi.fn()
  }
}))

mockNuxtImport('navigateTo', () => navigateTo)
mockNuxtImport('reportProblem', () => reportProblem)
mockNuxtImport('useApiClient', () => () => api)

const HASH = 'ab'.repeat(32)

const user = (): UserRef => ({
  object: 'user',
  id: '9',
  name: 'Ada',
  avatar: null,
  avatar_frame: null
})

const topic = (over: Partial<Topic> = {}): Topic => ({
  object: 'topic',
  id: '77',
  title: 'Hello',
  access_scope: 'public',
  author: user(),
  author_moemoepoint: 10,
  best_answer: null,
  bumped_at: '2026-02-01T00:00:00.000Z',
  category: 'galgame',
  comment_count: 0,
  content: { object: 'document', children: [] },
  cover_images: [],
  created_at: '2026-01-01T00:00:00.000Z',
  dislike_count: 0,
  edited_at: null,
  favorite_count: 0,
  hidden_by: null,
  is_nsfw: false,
  like_count: 0,
  mini_apps: [],
  pinned_reply: null,
  reactions: [],
  reply_count: 0,
  sections: ['g-chatting'],
  state: 'published',
  upvote_count: 0,
  upvoted_at: null,
  view_count: 1,
  viewer: null,
  ...over
})

const jsonResult = (status: number, body: unknown) => ({
  data: status >= 200 && status < 300 ? body : undefined,
  error: status >= 200 && status < 300 ? undefined : body,
  response: new Response(JSON.stringify(body), {
    status,
    headers: {
      'content-type':
        status >= 400 ? 'application/problem+json' : 'application/json'
    }
  })
})

const fillCreate = () => {
  const persist = usePersistEditTopicStore()
  persist.title = 'Hello'
  persist.content = 'Body of the topic'
  persist.category = 'galgame'
  persist.section = ['g-chatting']
  persist.isNSFW = true
  persist.coverImages = []
  persist.accessScope = 'public'
  persist.accessRoles = []
  persist.accessUserIds = []
  useTempEditStore().isTopicRewriting = false
  usePersistUserStore().moemoepoint = 100
}

afterEach(() => {
  api.POST.mockReset()
  api.PATCH.mockReset()
  navigateTo.mockReset()
  reportProblem.mockReset()
  usePersistEditTopicStore().resetTopicData()
  useTempEditStore().resetRewriteTopicData()
})

type Captured = {
  method: string
  path: string
  topicId?: string
  body: unknown
  key: string | null
}

type CallInit = {
  params?: {
    path?: { topic_id?: string }
    header?: { 'Idempotency-Key'?: string }
  }
  body?: unknown
}

const captureInit = (
  method: string,
  path: string,
  init: CallInit
): Captured => ({
  method,
  path,
  topicId: init.params?.path?.topic_id,
  body: init.body,
  key: init.params?.header?.['Idempotency-Key'] ?? null
})

describe('useTopicSubmitter', () => {
  it('creates without cover_image_hashes when none are picked, and always sends is_nsfw', async () => {
    const captured: Captured[] = []
    api.POST.mockImplementation(async (path: string, init: CallInit) => {
      captured.push(captureInit('POST', path, init))
      return jsonResult(201, topic())
    })
    fillCreate()
    const { submit } = useTopicSubmitter()
    await submit()
    expect(captured).toHaveLength(1)
    expect(captured[0]!.method).toBe('POST')
    expect(captured[0]!.path).toBe('/topics')
    expect(captured[0]!.body).toEqual({
      title: 'Hello',
      content_markdown: 'Body of the topic',
      category: 'galgame',
      sections: ['g-chatting'],
      is_nsfw: true,
      access_scope: 'public'
    })
    expect(captured[0]!.body).not.toHaveProperty('cover_image_hashes')
    expect(captured[0]!.key).toEqual(expect.any(String))
    expect(navigateTo).toHaveBeenCalledWith('/topic/77')
  })

  it('creates with cover hashes and grant fields for each access scope', async () => {
    const captured: Captured[] = []
    api.POST.mockImplementation(async (path: string, init: CallInit) => {
      captured.push(captureInit('POST', path, init))
      return jsonResult(201, topic())
    })
    fillCreate()
    const persist = usePersistEditTopicStore()
    persist.coverImages = [`/image/${HASH}`]
    persist.accessScope = 'role'
    persist.accessRoles = ['moderator']
    await useTopicSubmitter().submit()
    fillCreate()
    persist.coverImages = [`/image/${HASH}`]
    persist.accessScope = 'users'
    persist.accessUserIds = [2, 3]
    persist.accessRoles = ['admin']
    await useTopicSubmitter().submit()
    fillCreate()
    persist.coverImages = [`/image/${HASH}`]
    persist.accessScope = 'login'
    persist.accessRoles = ['admin']
    persist.accessUserIds = [2]
    await useTopicSubmitter().submit()
    expect(captured[0]!.body).toMatchObject({
      cover_image_hashes: [HASH],
      access_scope: 'role',
      access_roles: ['moderator'],
      is_nsfw: true
    })
    expect(captured[0]!.body).not.toHaveProperty('access_user_ids')
    expect(captured[1]!.body).toMatchObject({
      access_scope: 'users',
      access_user_ids: ['2', '3'],
      is_nsfw: true
    })
    expect(captured[1]!.body).not.toHaveProperty('access_roles')
    expect(captured[2]!.body).toMatchObject({
      access_scope: 'login',
      is_nsfw: true
    })
    expect(captured[2]!.body).not.toHaveProperty('access_roles')
    expect(captured[2]!.body).not.toHaveProperty('access_user_ids')
  })

  it('rewrites with every field and cover_image_hashes [] when the picker is empty', async () => {
    const captured: Captured[] = []
    api.PATCH.mockImplementation(async (path: string, init: CallInit) => {
      captured.push(captureInit('PATCH', path, init))
      return jsonResult(200, topic({ id: '42' }))
    })
    const temp = useTempEditStore()
    temp.id = 42
    temp.title = 'Rewritten'
    temp.content = 'New body'
    temp.category = 'technique'
    temp.section = ['t-web']
    temp.isNSFW = false
    temp.coverImages = []
    temp.accessScope = 'public'
    temp.isTopicRewriting = true
    usePersistUserStore().moemoepoint = 100
    const { submit } = useTopicSubmitter()
    await submit()
    expect(captured).toHaveLength(1)
    expect(captured[0]!.method).toBe('PATCH')
    expect(captured[0]!.path).toBe('/topics/{topic_id}')
    expect(captured[0]!.topicId).toBe('42')
    expect(captured[0]!.body).toEqual({
      title: 'Rewritten',
      content_markdown: 'New body',
      category: 'technique',
      sections: ['t-web'],
      is_nsfw: false,
      cover_image_hashes: [],
      access_scope: 'public'
    })
    expect(captured[0]!.key).toBeNull()
    expect(navigateTo).toHaveBeenCalledWith('/topic/42')
  })

  it('reuses the idempotency key on a same-payload retry and renews it for a changed payload', async () => {
    const captured: Captured[] = []
    let n = 0
    api.POST.mockImplementation(async (path: string, init: CallInit) => {
      captured.push(captureInit('POST', path, init))
      n++
      if (n <= 2) {
        throw new TypeError('Failed to fetch')
      }
      return jsonResult(201, topic())
    })
    fillCreate()
    const { submit } = useTopicSubmitter()
    await submit()
    await submit()
    usePersistEditTopicStore().title = 'Hello 2'
    await submit()
    expect(captured).toHaveLength(3)
    expect(captured[0]!.key).toBe(captured[1]!.key)
    expect(captured[2]!.key).not.toBe(captured[1]!.key)
  })

  it('reports a problem and does not navigate', async () => {
    api.POST.mockResolvedValue(
      jsonResult(429, {
        type: 'about:blank',
        title: 'Error',
        status: 429,
        code: 'TOPIC_DAILY_LIMIT_REACHED',
        request_id: 'req_1',
        errors: []
      })
    )
    fillCreate()
    const { submit } = useTopicSubmitter()
    await submit()
    expect(reportProblem).toHaveBeenCalledTimes(1)
    expect(navigateTo).not.toHaveBeenCalled()
  })
})

describe('applyTopicSource', () => {
  it('fills every editor store field from TopicSource, including grants and covers', () => {
    const source: TopicSource = {
      object: 'topic_source',
      topic_id: '88',
      title: 'Source title',
      content_markdown: 'Source body',
      category: 'others',
      sections: ['o-daily', 'o-essay'],
      is_nsfw: true,
      access_scope: 'users',
      cover_images: [
        {
          hash: HASH,
          url: 'https://cdn.example/x.webp',
          width: 10,
          height: 10,
          thumbhash: null,
          sexual: null
        }
      ],
      access_grants: {
        roles: ['ren'],
        users: [
          { object: 'user', id: '5', name: 'Eve', avatar: null, avatar_frame: null },
          { object: 'user', id: '6', name: null, avatar: null, avatar_frame: null }
        ]
      }
    }
    applyTopicSource(source)
    const temp = useTempEditStore()
    expect(temp.id).toBe(88)
    expect(temp.title).toBe('Source title')
    expect(temp.content).toBe('Source body')
    expect(temp.category).toBe('others')
    expect(temp.section).toEqual(['o-daily', 'o-essay'])
    expect(temp.isNSFW).toBe(true)
    expect(temp.coverImages).toEqual([`/image/${HASH}`])
    expect(temp.accessScope).toBe('users')
    expect(temp.accessRoles).toEqual(['ren'])
    expect(temp.accessUserIds).toEqual([5, 6])
    expect(temp.accessUsers).toEqual([
      { id: 5, name: 'Eve', avatar: '', avatarDecoration: null },
      { id: 6, name: '已注销用户', avatar: '', avatarDecoration: null }
    ])
    expect(temp.isTopicRewriting).toBe(true)
  })
})
