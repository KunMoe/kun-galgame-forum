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
mockNuxtImport('useTopicSections', () => () => ({
  sectionOf: (key?: string | null) =>
    key && ['galgame-discussion', 'development', 'daily'].includes(key)
      ? { section: key, is_archived: false, moemoepoint_on_create: 3 }
      : undefined
}))

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
  section: 'galgame-discussion',
  ai_summary: null,
  sections: ['galgame-discussion'],
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
  persist.section = ['galgame-discussion']
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
      sections: ['galgame-discussion'],
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
    temp.section = ['development']
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
      sections: ['development'],
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

  it('creates attached polls and lotteries on the new topic and holds the ones that fail', async () => {
    const captured: Captured[] = []
    api.POST.mockImplementation(async (path: string, init: CallInit) => {
      captured.push(captureInit('POST', path, init))
      if (path === '/topics/{topic_id}/lotteries') {
        return jsonResult(403, {
          type: 'about:blank',
          title: 'Error',
          status: 403,
          code: 'LOTTERY_CREATOR_INELIGIBLE',
          request_id: 'req_1',
          errors: []
        })
      }
      return jsonResult(201, path === '/topics' ? topic() : {})
    })
    fillCreate()
    const persist = usePersistEditTopicStore()
    persist.polls = [
      {
        title: 'Route',
        description: '',
        options: [{ text: 'a' }, { text: 'b' }],
        choice_type: 'single',
        min_choice: 1,
        max_choice: 1,
        closes_at: undefined,
        result_visibility: 'always',
        is_anonymous: false,
        can_change_vote: true
      }
    ]
    persist.lotteries = [
      {
        title: 'Keys',
        description: '',
        entry_mode: 'signup',
        floor_rule: '',
        draw_mode: 'manual',
        draw_threshold: 0,
        deadline: undefined,
        min_account_age_days: 0,
        min_moemoepoint: 0,
        show_entrants: true,
        prizes: [
          {
            name: 'Key',
            description: '',
            image_hashes: [],
            image_urls: [],
            nsfw_hashes: [],
            machine_nsfw_hashes: [],
            delivery: 'code',
            point_mode: 'fixed',
            point_amount: 0,
            slots: 1,
            codes: 'SECRET'
          }
        ]
      }
    ]
    await useTopicSubmitter().submit()
    expect(captured.map((c) => c.path)).toEqual([
      '/topics',
      '/topics/{topic_id}/polls',
      '/topics/{topic_id}/lotteries'
    ])
    expect(captured[1]!.topicId).toBe('77')
    expect(captured[1]!.key).not.toBe(captured[0]!.key)
    expect(captured[2]!.body).toMatchObject({
      prizes: [{ codes: ['SECRET'] }]
    })
    expect(persist.polls).toEqual([])
    expect(persist.lotteries).toEqual([])
    const held = useTempTopicMiniAppHandoffStore().heldFor('77')
    expect(held).toHaveLength(1)
    expect(held[0]!.key).toBe('lottery')
    expect(reportProblem).toHaveBeenCalledTimes(1)
    expect(navigateTo).toHaveBeenCalledWith('/topic/77')
  })

  it('does not create the topic when an attached lottery would be refused', async () => {
    fillCreate()
    usePersistEditTopicStore().lotteries = [
      {
        title: 'Late',
        description: '',
        entry_mode: 'signup',
        floor_rule: '',
        draw_mode: 'deadline',
        draw_threshold: 0,
        deadline: '2001-01-01',
        min_account_age_days: 0,
        min_moemoepoint: 0,
        show_entrants: true,
        prizes: []
      }
    ]
    await useTopicSubmitter().submit()
    expect(api.POST).not.toHaveBeenCalled()
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
      section: 'daily',
      sections: ['daily'],
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
    expect(temp.section).toEqual(['daily'])
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
