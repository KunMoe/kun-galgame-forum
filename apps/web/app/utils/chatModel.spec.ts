import { describe, expect, it } from 'vitest'
import type {
  ChatConversation,
  ChatMessage,
  ChatPush,
  ChatUpdate
} from '#shared/types/chat'
import fixture from '~~/tests/fixtures/chat.json'
import {
  applyChatUpdate,
  chatFolderOf,
  compareChatConversations,
  upsertChatMessage,
  type ChatModel,
  type ChatWindow
} from './chatModel'

const clone = <T>(v: unknown): T => structuredClone(v) as T
const push = (p: unknown) => clone<Extract<ChatPush, { type: 'update' }>>(p)

const modelWith = (
  patch: (c: ChatConversation) => void = () => {},
  items: ChatMessage[] = []
): ChatModel => {
  const c = clone<ChatConversation>(fixture.conversation)
  patch(c)
  return {
    me: '2',
    conversations: { [c.id]: c },
    windows: { [c.id]: { items, hasOlder: false, hasNewer: false } },
    users: {}
  }
}

const conv = (m: ChatModel) => m.conversations['3']!
const win = (m: ChatModel) => m.windows['3']!

describe('applyChatUpdate on real pushes', () => {
  it("appends the peer's new message and counts it unread", () => {
    const p = push(fixture.push.newMessage)
    const m = modelWith((c) => {
      c.last_seq = 0
      c.last_message = null
      c.me.last_read_seq = 0
      c.me.unread_count = 0
    })
    expect(applyChatUpdate(m, p.update, { message: p.message })).toBe('applied')
    expect(win(m).items.map((x) => x.seq)).toEqual([p.message!.seq])
    expect(conv(m).last_message?.id).toBe(p.message!.id)
    expect(conv(m).me.unread_count).toBe(1)
  })

  it('does not count a message at or below the read position twice', () => {
    const p = push(fixture.push.newMessage)
    const m = modelWith((c) => {
      c.me.last_read_seq = p.message!.seq
      c.me.unread_count = 0
    })
    applyChatUpdate(m, p.update, { message: p.message })
    expect(conv(m).me.unread_count).toBe(0)
  })

  it('an own message reads everything for the sender', () => {
    const p = push(fixture.push.ownNewMessage)
    const m = modelWith((c) => {
      c.me.unread_count = 4
      c.me.marked_unread = true
      c.me.last_read_seq = 0
    })
    applyChatUpdate(m, p.update, { message: p.message })
    expect(conv(m).me).toMatchObject({
      unread_count: 0,
      marked_unread: false,
      last_read_seq: p.message!.seq
    })
  })

  it('a confirmed send replaces its pending copy by client_message_id', () => {
    const p = push(fixture.push.ownNewMessage)
    const pending: ChatMessage = {
      ...p.message!,
      id: '',
      seq: 0,
      status: 'sending'
    }
    const w: ChatWindow = { items: [pending], hasOlder: false, hasNewer: false }
    upsertChatMessage(w, p.message!)
    expect(w.items).toHaveLength(1)
    expect(w.items[0]!.status).toBeUndefined()
    expect(w.items[0]!.seq).toBe(p.message!.seq)
  })

  it('read_outbox moves the double tick', () => {
    const p = push(fixture.push.readOutbox)
    const m = modelWith((c) => (c.peer_read_seq = 0))
    applyChatUpdate(m, p.update)
    expect(conv(m).peer_read_seq).toBe(Number(p.update.data.max_seq))
  })

  it('accepting moves a request into the inbox', () => {
    const p = push(fixture.push.dialogAccepted)
    const m = modelWith((c) => (c.me.accepted = false))
    expect(chatFolderOf(conv(m))).toBe('requests')
    applyChatUpdate(m, p.update)
    expect(chatFolderOf(conv(m))).toBe('inbox')
  })

  it('reactions come with the push and replace the counts', () => {
    const p = push(fixture.push.reactions)
    const target = clone<ChatMessage[]>(fixture.messagePage.items).map((x) => ({
      ...x,
      seq: Number(p.update.data.seq),
      reactions: []
    }))[0]!
    const m = modelWith(undefined, [target])
    applyChatUpdate(m, p.update, { reactions: p.reactions })
    expect(win(m).items[0]!.reactions).toEqual(p.reactions)
  })

  it('an edit replaces the loaded copy and the pinned one', () => {
    const p = push(fixture.push.editMessage)
    const stale = { ...p.message!, text: 'before', edited_at: null }
    const m = modelWith((c) => (c.pinned_messages = [stale]), [stale])
    applyChatUpdate(m, p.update, { message: p.message })
    expect(win(m).items[0]!.text).toBe(p.message!.text)
    expect(conv(m).pinned_messages![0]!.text).toBe(p.message!.text)
  })

  it('a delete drops the seqs and asks for the conversation again', () => {
    const p = push(fixture.push.deleteMessages)
    const seq = (p.update.data.seqs as number[])[0]!
    const base = clone<ChatMessage[]>(fixture.messagePage.items)[0]!
    const m = modelWith(undefined, [
      { ...base, seq: seq - 1 },
      { ...base, seq }
    ])
    expect(applyChatUpdate(m, p.update)).toBe('refetch')
    expect(win(m).items.map((x) => x.seq)).toEqual([seq - 1])
  })

  it('an update for a conversation not held asks for it', () => {
    const p = push(fixture.push.newMessage)
    const m: ChatModel = { me: '2', conversations: {}, windows: {}, users: {} }
    expect(applyChatUpdate(m, p.update, { message: p.message })).toBe('refetch')
    const cleared: ChatUpdate = {
      ...p.update,
      kind: 'clear_history',
      data: { through_seq: 3, removed: true }
    }
    expect(applyChatUpdate(m, cleared)).toBe('applied')
  })
})

describe('the update stream page', () => {
  it('replays in order onto the conversation it names', () => {
    const page = fixture.updatesPage
    const m = modelWith((c) => {
      c.last_seq = 0
      c.last_message = null
      c.peer_read_seq = 0
      c.me.last_read_seq = 0
      c.me.unread_count = 0
      c.me.accepted = false
    })
    const byKey = new Map(
      (page.messages as ChatMessage[]).map((x) => [
        `${x.conversation_id}:${x.seq}`,
        x
      ])
    )
    for (const u of page.updates as ChatUpdate[]) {
      applyChatUpdate(m, u, {
        message: byKey.get(`${u.conversation_id}:${u.data.seq}`)
      })
    }
    expect(conv(m).me.accepted).toBe(true)
    expect(conv(m).me.unread_count).toBe(0)
    expect(conv(m).peer_read_seq).toBeGreaterThan(0)
    expect(win(m).items.map((x) => x.seq)).toEqual(
      [...win(m).items.map((x) => x.seq)].sort((a, b) => a - b)
    )
  })
})

describe('compareChatConversations', () => {
  it('puts pinned conversations first, then the latest activity', () => {
    const base = clone<ChatConversation>(fixture.conversation)
    const old = { ...base, id: 'a', me: { ...base.me, pinned_rank: null } }
    const recent = {
      ...base,
      id: 'b',
      me: { ...base.me, pinned_rank: null },
      last_message: {
        ...base.last_message!,
        created_at: '2099-01-01T00:00:00Z'
      }
    }
    const pinned = { ...base, id: 'c', me: { ...base.me, pinned_rank: 1 } }
    expect(
      [old, recent, pinned].sort(compareChatConversations).map((c) => c.id)
    ).toEqual(['c', 'b', 'a'])
  })
})
