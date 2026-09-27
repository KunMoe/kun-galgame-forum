import type { KunChatFormattedText, KunChatReplyQuote } from '@kungal/ui-vue'
import type {
  ChatConversationDetail,
  ChatDeleted,
  ChatDialog,
  ChatMessage,
  ChatPhoto,
  ChatReactionsBody,
  ChatReadState,
  ChatReportReason,
  WithUsers
} from '#shared/types/chat'
import type { ApiResult } from '#shared/utils/api/problem'
import {
  mergeChatUsers,
  replaceChatMessage,
  upsertChatMessage
} from '~/utils/chatModel'

const TYPING_EVERY = 5000
const typingSentAt: Record<string, number> = {}

export interface ChatSendOptions {
  replyTo?: ChatMessage | null
  quote?: KunChatReplyQuote | null
  photo?: Pick<ChatPhoto, 'image_hash' | 'width' | 'height' | 'thumbhash'>
}

export const useChatActions = () => {
  const api = useChatApi()
  const store = useChatStore()
  const model = store.model

  const settled = <T>(r: ApiResult<T>) => {
    if (!r.ok) {
      store.failed(r.problem)
    }
    return r
  }

  const post = async (id: string, pending: ChatMessage) => {
    const body: Record<string, unknown> = {
      client_message_id: pending.client_message_id,
      text: pending.text,
      entities: pending.entities
    }
    if (pending.reply_to) {
      body.reply_to_seq = pending.reply_to.seq
    }
    if (pending.reply_quote) {
      body.reply_quote = {
        text: pending.reply_quote.text,
        offset: pending.reply_quote.offset
      }
    }
    if (pending.media) {
      body.media = { type: 'photo', image_hash: pending.media.image_hash }
    }
    const r = settled(
      await api.post<WithUsers<ChatMessage>>(
        `/conversations/${id}/messages`,
        body
      )
    )
    const w = model.windows[id]
    if (!r.ok) {
      const at =
        w?.items.findIndex(
          (m) => m.client_message_id === pending.client_message_id
        ) ?? -1
      if (w && at >= 0) {
        w.items[at] = { ...w.items[at]!, status: 'failed' }
      }
      return r
    }
    const { users, ...msg } = r.data
    mergeChatUsers(model, users)
    if (w) {
      upsertChatMessage(w, msg)
    }
    const c = model.conversations[id]
    if (c && (!c.last_message || msg.seq >= c.last_message.seq)) {
      c.last_message = msg
      c.last_seq = Math.max(c.last_seq, msg.seq)
    }
    return r
  }

  const send = (
    id: string,
    text: KunChatFormattedText,
    o: ChatSendOptions = {}
  ) => {
    const replyTo = o.replyTo
    const pending: ChatMessage = {
      id: '',
      conversation_id: id,
      seq: 0,
      sender_id: model.me,
      kind: 'message',
      text: text.text,
      entities: text.entities,
      media: o.photo ? { type: 'photo', ...o.photo } : null,
      media_group_id: null,
      reply_to: replyTo
        ? {
            seq: replyTo.seq,
            sender_id: replyTo.sender_id,
            text: replyTo.text,
            entities: replyTo.entities,
            media_type: replyTo.media?.type ?? null,
            deleted: false
          }
        : null,
      reply_quote: o.quote ?? null,
      service_action: null,
      context: null,
      reactions: [],
      silent: false,
      pinned_at: null,
      edited_at: null,
      created_at: new Date().toISOString(),
      client_message_id: crypto.randomUUID(),
      status: 'sending'
    }
    model.windows[id]?.items.push(pending)
    return post(id, pending)
  }

  const retry = (message: ChatMessage) => {
    const w = model.windows[message.conversation_id]
    const at = w?.items.indexOf(message) ?? -1
    if (!w || at < 0) {
      return
    }
    w.items[at] = { ...message, status: 'sending' }
    return post(message.conversation_id, w.items[at]!)
  }

  const discard = (message: ChatMessage) => {
    const w = model.windows[message.conversation_id]
    if (w) {
      w.items = w.items.filter((m) => m !== message)
    }
  }

  const replace = (msg: ChatMessage) => {
    const w = model.windows[msg.conversation_id]
    if (w) {
      replaceChatMessage(w, msg)
    }
  }

  const edit = async (message: ChatMessage, text: KunChatFormattedText) => {
    const r = settled(
      await api.patch<WithUsers<ChatMessage>>(`/messages/${message.id}`, text)
    )
    if (r.ok) {
      const { users, ...msg } = r.data
      mergeChatUsers(model, users)
      replace(msg)
    }
    return r
  }

  const remove = async (id: string, seqs: number[], forEveryone: boolean) => {
    const r = settled(
      await api.post<ChatDeleted>(`/conversations/${id}/messages/delete`, {
        seqs,
        for_everyone: forEveryone
      })
    )
    if (r.ok) {
      const gone = new Set(r.data.seqs)
      const w = model.windows[id]
      if (w) {
        w.items = w.items.filter((m) => m.seq === 0 || !gone.has(m.seq))
      }
    }
    return r
  }

  const react = async (message: ChatMessage, reaction: string | null) => {
    const r = settled(
      await api.put<ChatReactionsBody>(`/messages/${message.id}/reaction`, {
        reaction
      })
    )
    if (r.ok) {
      replace({ ...message, reactions: r.data.reactions })
    }
    return r
  }

  const pin = async (id: string, target: number, pinned: boolean) => {
    const path = `/conversations/${id}/pins/${target}`
    const r = settled(pinned ? await api.put(path) : await api.del(path))
    if (r.ok) {
      void store.refetch(id)
    }
    return r
  }

  const read = async (id: string, maxSeq: number) => {
    const c = model.conversations[id]
    if (!c || (maxSeq <= c.me.last_read_seq && !c.me.marked_unread)) {
      return
    }
    const r = settled(
      await api.post<ChatReadState>(`/conversations/${id}/read`, {
        max_seq: maxSeq
      })
    )
    if (r.ok) {
      c.me.last_read_seq = Math.max(c.me.last_read_seq, r.data.last_read_seq)
      c.me.unread_count = r.data.unread_count
      c.me.marked_unread = false
      store.refreshState()
    }
  }

  const accept = async (id: string) => {
    const r = settled(await api.post(`/conversations/${id}/accept`))
    if (r.ok) {
      await store.refetch(id)
      store.refreshState()
    }
    return r
  }

  const setDialog = async (id: string, patch: Record<string, unknown>) => {
    const r = settled(
      await api.patch<ChatDialog & { object: 'dialog_state' }>(
        `/conversations/${id}/me`,
        patch
      )
    )
    const c = model.conversations[id]
    if (r.ok && c) {
      const { object: _object, ...dialog } = r.data
      Object.assign(c.me, dialog)
      store.refreshState()
    }
    return r
  }

  const clearHistory = async (id: string, remove: boolean) => {
    const r = settled(
      await api.post(`/conversations/${id}/clear-history`, { remove })
    )
    if (r.ok) {
      store.refreshState()
    }
    return r
  }

  const report = async (
    message: ChatMessage,
    reason: ChatReportReason,
    note?: string
  ) =>
    settled(
      await api.post(`/messages/${message.id}/report`, {
        reason,
        ...(note ? { note } : {})
      })
    )

  const saveDraft = async (
    id: string,
    text: KunChatFormattedText,
    replyToSeq: number | null
  ) =>
    settled(
      await api.put(`/conversations/${id}/draft`, {
        ...text,
        ...(replyToSeq ? { reply_to_seq: replyToSeq } : {})
      })
    )

  const sendTyping = (id: string) => {
    const now = Date.now()
    if (now - (typingSentAt[id] ?? 0) < TYPING_EVERY) {
      return
    }
    typingSentAt[id] = now
    void api.post(`/conversations/${id}/typing`)
  }

  const openDirect = async (userId: number | string) => {
    const r = settled(
      await api.put<ChatConversationDetail>(`/direct/${userId}`)
    )
    if (r.ok) {
      store.putConversation(r.data)
    }
    return r
  }

  const uploadPhoto = async (file: File) => {
    const form = new FormData()
    form.append('file', file)
    return settled(await api.post<ChatPhoto>('/images', form))
  }

  return {
    send,
    retry,
    discard,
    edit,
    remove,
    react,
    pin,
    read,
    accept,
    setDialog,
    clearHistory,
    report,
    saveDraft,
    sendTyping,
    openDirect,
    uploadPhoto
  }
}
