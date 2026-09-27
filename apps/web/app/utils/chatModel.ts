import type { KunChatReaction } from '@kungal/ui-vue'
import type {
  ChatConversation,
  ChatFolder,
  ChatMessage,
  ChatUpdate,
  ChatUser
} from '#shared/types/chat'

export interface ChatWindow {
  items: ChatMessage[]
  hasOlder: boolean
  hasNewer: boolean
}

export interface ChatModel {
  me: string
  conversations: Record<string, ChatConversation>
  windows: Record<string, ChatWindow>
  users: Record<string, ChatUser>
}

export interface ChatUpdateExtras {
  message?: ChatMessage
  reactions?: KunChatReaction[]
}

export type ChatApplyOutcome = 'applied' | 'refetch' | 'removed'

const seqsOf = (data: Record<string, unknown>): number[] =>
  Array.isArray(data.seqs) ? data.seqs.map(Number) : []

export const isPendingChatMessage = (m: ChatMessage) =>
  m.status === 'sending' || m.status === 'failed'

export const chatActivityAt = (c: ChatConversation): string =>
  c.last_message?.created_at ?? c.created_at

export const chatMuted = (c: ChatConversation, now = Date.now()): boolean =>
  !!c.me.muted_until && Date.parse(c.me.muted_until) > now

export const chatFolderOf = (c: ChatConversation): ChatFolder =>
  !c.me.accepted ? 'requests' : c.me.archived ? 'archive' : 'inbox'

export const compareChatConversations = (
  a: ChatConversation,
  b: ChatConversation
): number => {
  const ra = a.me.pinned_rank ?? -1
  const rb = b.me.pinned_rank ?? -1
  if (ra !== rb) {
    return rb - ra
  }
  return chatActivityAt(b).localeCompare(chatActivityAt(a))
}

export const mergeChatUsers = (m: ChatModel, users: ChatUser[] = []) => {
  for (const u of users) {
    m.users[u.id] = u
  }
}

export const upsertChatMessage = (w: ChatWindow, msg: ChatMessage) => {
  const byClient = msg.client_message_id
    ? w.items.findIndex(
        (x) =>
          x.client_message_id === msg.client_message_id &&
          isPendingChatMessage(x)
      )
    : -1
  if (byClient >= 0) {
    w.items.splice(byClient, 1)
  }
  const at = w.items.findIndex(
    (x) => !isPendingChatMessage(x) && x.seq === msg.seq
  )
  if (at >= 0) {
    w.items[at] = {
      ...msg,
      client_message_id: w.items[at]!.client_message_id ?? msg.client_message_id
    }
    return
  }
  const confirmed = w.items.filter((x) => !isPendingChatMessage(x))
  const last = confirmed[confirmed.length - 1]
  if (w.hasNewer && (!last || msg.seq > last.seq)) {
    return
  }
  let i = w.items.length
  while (
    i > 0 &&
    (isPendingChatMessage(w.items[i - 1]!) || w.items[i - 1]!.seq > msg.seq)
  ) {
    i--
  }
  w.items.splice(i, 0, msg)
}

export const replaceChatMessage = (w: ChatWindow, msg: ChatMessage) => {
  const at = w.items.findIndex(
    (x) => !isPendingChatMessage(x) && x.seq === msg.seq
  )
  if (at >= 0) {
    w.items[at] = { ...msg, client_message_id: w.items[at]!.client_message_id }
  }
}

const dropSeqs = (w: ChatWindow | undefined, seqs: number[]) => {
  if (!w || !seqs.length) {
    return
  }
  const gone = new Set(seqs)
  w.items = w.items.filter((x) => isPendingChatMessage(x) || !gone.has(x.seq))
}

export const applyChatUpdate = (
  m: ChatModel,
  u: ChatUpdate,
  x: ChatUpdateExtras = {}
): ChatApplyOutcome => {
  const c = m.conversations[u.conversation_id]
  if (!c) {
    return u.kind === 'clear_history' ? 'applied' : 'refetch'
  }
  const w = m.windows[c.id]
  const d = u.data
  switch (u.kind) {
    case 'new_message': {
      const seq = Number(d.seq)
      c.last_seq = Math.max(c.last_seq, seq)
      const msg = x.message
      if (!msg) {
        return 'applied'
      }
      if (w) {
        upsertChatMessage(w, msg)
      }
      if (!c.last_message || msg.seq >= c.last_message.seq) {
        c.last_message = msg
      }
      if (msg.sender_id === m.me) {
        c.me.last_read_seq = Math.max(c.me.last_read_seq, seq)
        c.me.unread_count = 0
        c.me.marked_unread = false
      } else if (msg.kind === 'message' && seq > c.me.last_read_seq) {
        c.me.unread_count += 1
      }
      return 'applied'
    }
    case 'edit_message': {
      const msg = x.message
      if (!msg) {
        return 'applied'
      }
      if (w) {
        replaceChatMessage(w, msg)
      }
      if (c.last_message?.seq === msg.seq) {
        c.last_message = msg
      }
      c.pinned_messages = c.pinned_messages?.map((p) =>
        p.seq === msg.seq ? msg : p
      )
      return 'applied'
    }
    case 'delete_messages':
      dropSeqs(w, seqsOf(d))
      return 'refetch'
    case 'hide_messages': {
      const seqs = seqsOf(d)
      dropSeqs(w, seqs)
      c.me.unread_count = Number(d.unread_count ?? c.me.unread_count)
      return c.last_message && seqs.includes(c.last_message.seq)
        ? 'refetch'
        : 'applied'
    }
    case 'read_inbox':
      c.me.last_read_seq = Math.max(c.me.last_read_seq, Number(d.max_seq))
      c.me.unread_count = Number(d.unread_count ?? 0)
      return 'applied'
    case 'read_outbox':
      c.peer_read_seq = Math.max(c.peer_read_seq, Number(d.max_seq))
      return 'applied'
    case 'message_reactions': {
      const reactions = x.reactions ?? x.message?.reactions
      const target = w?.items.find(
        (i) => !isPendingChatMessage(i) && i.seq === Number(d.seq)
      )
      if (target && reactions) {
        target.reactions = reactions
      }
      return 'applied'
    }
    case 'pinned_messages': {
      const seqs = seqsOf(d)
      const pinned = d.pinned === true
      for (const i of w?.items ?? []) {
        if (!isPendingChatMessage(i) && seqs.includes(i.seq)) {
          i.pinned_at = pinned ? u.created_at : null
        }
      }
      return 'refetch'
    }
    case 'dialog':
      Object.assign(c.me, d)
      return 'applied'
    case 'clear_history': {
      const through = Number(d.through_seq)
      c.me.cleared_through_seq = Math.max(c.me.cleared_through_seq, through)
      if (w) {
        w.items = w.items.filter(
          (i) => isPendingChatMessage(i) || i.seq > through
        )
      }
      if (c.last_message && c.last_message.seq <= through) {
        c.last_message = null
      }
      if (d.removed === true) {
        Reflect.deleteProperty(m.conversations, c.id)
        Reflect.deleteProperty(m.windows, c.id)
        return 'removed'
      }
      return 'applied'
    }
    case 'member':
      if (d.action === 'deleted' && typeof d.user_id === 'string') {
        m.users[d.user_id] = {
          id: d.user_id,
          name: '',
          avatar: '',
          deleted: true
        }
      }
      return 'refetch'
    case 'conversation':
      Object.assign(c, d)
      return 'applied'
  }
}
