import { describe, expect, it, vi } from 'vitest'
import type { Topic, UserRef } from '#shared/utils/api/schemas'
import { createApiClient } from '#shared/utils/api/client'
import { fetchTopicOgCard, topicToOgCard } from './kunOgCard'

const user = (over: Partial<UserRef> = {}): UserRef => ({
  object: 'user',
  id: '8',
  name: 'Neko',
  avatar: {
    hash: 'ab',
    url: 'https://cdn.example/ab.webp',
    width: 64,
    height: 64,
    thumbhash: null,
    sexual: null
  },
  avatar_frame: null,
  ...over
})

const topic = (over: Partial<Topic> = {}): Topic => ({
  object: 'topic',
  id: '42',
  title: 'A topic title',
  access_scope: 'public',
  author: user(),
  author_moemoepoint: 12,
  best_answer: null,
  bumped_at: '2026-02-01T00:00:00.000Z',
  category: 'galgame',
  comment_count: 0,
  content: {
    object: 'document',
    children: [
      {
        object: 'paragraph',
        children: [{ object: 'text', value: 'plain excerpt' }]
      }
    ]
  },
  cover_images: [],
  created_at: '2026-01-01T00:00:00.000Z',
  dislike_count: 0,
  edited_at: null,
  favorite_count: 0,
  hidden_by: null,
  is_nsfw: false,
  like_count: 7,
  mini_apps: [],
  pinned_reply: null,
  reactions: [],
  reply_count: 4,
  sections: ['g-chatting'],
  state: 'published',
  upvote_count: 1,
  upvoted_at: null,
  view_count: 88,
  viewer: null,
  ...over
})

const jsonResponse = (status: number, body: unknown, contentType: string) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { 'content-type': contentType }
  })

describe('topicToOgCard', () => {
  it('returns null when the topic is NSFW', () => {
    expect(topicToOgCard(topic({ is_nsfw: true }))).toBeNull()
  })

  it('returns null when the topic is hidden', () => {
    expect(
      topicToOgCard(topic({ state: 'hidden', hidden_by: 'author' }))
    ).toBeNull()
  })

  it('maps v1 fields onto the topic card', () => {
    expect(topicToOgCard(topic())).toEqual({
      template: 'topic',
      fields: {
        title: 'A topic title',
        excerpt: 'plain excerpt',
        section: '闲聊',
        author: 'Neko',
        authorAvatar: 'https://cdn.example/ab.webp',
        views: 88,
        replies: 4,
        likes: 7
      }
    })
  })

  it('uses the deleted-user label when the author name is null', () => {
    const card = topicToOgCard(topic({ author: user({ name: null }) }))
    expect(card?.fields.author).toBe('已注销用户')
  })
})

describe('fetchTopicOgCard', () => {
  it('returns null on 404', async () => {
    const fetchSpy = vi.fn(async (_input: Request) =>
      jsonResponse(
        404,
        {
          type: 'about:blank',
          title: 'Not found',
          status: 404,
          code: 'NOT_FOUND',
          request_id: 'req_01ARZ3NDEKTSV4RRFFQ69G5FAV',
          errors: []
        },
        'application/problem+json'
      )
    )
    const api = createApiClient({
      origin: 'http://og.test',
      fetch: fetchSpy,
      timeoutMs: 1000
    })
    expect(await fetchTopicOgCard(api, '42')).toBeNull()
    expect(fetchSpy).toHaveBeenCalledTimes(1)
    const [first] = fetchSpy.mock.calls
    expect(first?.[0]).toBeInstanceOf(Request)
    const url = new URL((first![0] as Request).url)
    expect(url.pathname.endsWith('/topics/42')).toBe(true)
  })
})
