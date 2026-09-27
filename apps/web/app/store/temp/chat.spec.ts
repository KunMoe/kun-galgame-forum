// @vitest-environment nuxt
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import type { ChatPush, ChatUpdate } from '#shared/types/chat'
import fixture from '~~/tests/fixtures/chat.json'
import { useChatStore } from './chat'

type Reply = { status?: number; body?: unknown }

let calls: string[] = []
let routes: Record<string, () => Reply> = {}

const json = (r: Reply) =>
  new Response(r.body === undefined ? null : JSON.stringify(r.body), {
    status: r.status ?? 200,
    headers: {
      'Content-Type':
        (r.status ?? 200) >= 400
          ? 'application/problem+json'
          : 'application/json'
    }
  })

const state = (lastUpdateSeq: number) => ({
  object: 'chat_state',
  last_update_seq: lastUpdateSeq,
  unread_conversation_count: 0,
  unread_message_count: 0,
  request_count: 0
})

const readOutbox = (updateSeq: number, maxSeq: number): ChatUpdate => ({
  ...(structuredClone(fixture.push.readOutbox.update) as ChatUpdate),
  update_seq: updateSeq,
  data: { max_seq: maxSeq }
})

const pushOf = (u: ChatUpdate): ChatPush => ({ type: 'update', update: u })

const updatesPage = (updates: ChatUpdate[], extra = {}) => ({
  object: 'update_list',
  updates,
  messages: [],
  conversations: [],
  users: [],
  last_update_seq: updates.at(-1)?.update_seq ?? 0,
  has_more: false,
  too_long: false,
  ...extra
})

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))

beforeEach(() => {
  setActivePinia(createPinia())
  calls = []
  routes = {
    'chat/state': () => ({ body: state(7) }),
    'chat/conversations': () => ({ body: fixture.conversationPage }),
    'chat/reactions': () => ({ body: { object: 'list', items: [] } }),
    'me/notification-preferences': () => ({
      body: { object: 'notification_preferences', muted_types: [] }
    })
  }
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL) => {
      const url = new URL(input instanceof Request ? input.url : String(input))
      const path = url.pathname.split('/').slice(3).join('/')
      calls.push(path + url.search)
      const route = routes[path]
      return json(
        route
          ? route()
          : { status: 404, body: { status: 404, code: 'NOT_FOUND' } }
      )
    })
  )
})

afterEach(() => {
  vi.unstubAllGlobals()
})

const started = async () => {
  const chat = useChatStore()
  await chat.start('2')
  expect(chat.status).toBe('ready')
  return chat
}

const peerRead = () => useChatStore().model.conversations['3']!.peer_read_seq

describe('chat store sync', () => {
  it('holds an early update until the missing one arrives, without a fetch', async () => {
    const chat = await started()
    chat.push(pushOf(readOutbox(9, 900)))
    expect(peerRead()).not.toBe(900)
    chat.push(pushOf(readOutbox(8, 800)))
    expect(peerRead()).toBe(900)
    await sleep(700)
    expect(calls.some((c) => c.startsWith('chat/updates'))).toBe(false)
  })

  it('reads the stream after half a second when the gap stays open', async () => {
    const chat = await started()
    routes['chat/updates'] = () => ({
      body: updatesPage([readOutbox(8, 800), readOutbox(9, 900)])
    })
    chat.push(pushOf(readOutbox(9, 900)))
    await sleep(200)
    expect(calls.some((c) => c.startsWith('chat/updates'))).toBe(false)
    await sleep(600)
    expect(calls).toContain('chat/updates?after=7&limit=100')
    expect(peerRead()).toBe(900)
  })

  it('ignores an update it already applied', async () => {
    const chat = await started()
    chat.push(pushOf(readOutbox(8, 800)))
    chat.push(pushOf(readOutbox(8, 5)))
    expect(peerRead()).toBe(800)
  })

  it('reloads state and the list when the position is too old', async () => {
    const chat = await started()
    routes['chat/updates'] = () => ({
      body: updatesPage([], { too_long: true })
    })
    routes['chat/state'] = () => ({ body: state(40) })
    const before = calls.filter((c) => c.startsWith('chat/state')).length
    await chat.sync()
    expect(calls.filter((c) => c.startsWith('chat/state')).length).toBe(
      before + 1
    )
    chat.push(pushOf(readOutbox(41, 4100)))
    expect(peerRead()).toBe(4100)
  })

  it('stops at a scope problem and does not keep asking', async () => {
    routes['chat/state'] = () => ({
      status: 403,
      body: { status: 403, code: 'SCOPE_REQUIRED', errors: [] }
    })
    const chat = useChatStore()
    await chat.start('2')
    expect(chat.status).toBe('scope')
    expect(calls.filter((c) => c.startsWith('chat/')).length).toBe(1)
  })

  it('lights the dot for unread or requests unless direct messages are muted', async () => {
    routes['chat/state'] = () => ({
      body: { ...state(7), request_count: 1 }
    })
    const chat = await started()
    expect(chat.hasUnread).toBe(true)
    chat.muted = true
    expect(chat.hasUnread).toBe(false)
  })
})
