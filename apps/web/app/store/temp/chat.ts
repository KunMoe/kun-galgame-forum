import { defineStore } from 'pinia'
import type { KunChatReactionOption, KunChatTypingEvent } from '@kungal/ui-vue'
import type {
  ChatConversation,
  ChatConversationDetail,
  ChatConversationPage,
  ChatFolder,
  ChatMessagePage,
  ChatPush,
  ChatReactionList,
  ChatState,
  ChatUpdate,
  ChatUpdateKind,
  ChatUpdatesPage,
  WithUsers
} from '#shared/types/chat'
import type { ClientProblem } from '#shared/utils/api/problem'
import { settle } from '#shared/utils/api/problem'
import {
  applyChatUpdate,
  chatFolderOf,
  compareChatConversations,
  isPendingChatMessage,
  mergeChatUsers,
  upsertChatMessage,
  type ChatModel,
  type ChatUpdateExtras
} from '~/utils/chatModel'

const PAGE = 50
const GAP_WAIT = 500
const TYPING_TTL = 6000

const COUNTED: ReadonlySet<ChatUpdateKind> = new Set([
  'new_message',
  'read_inbox',
  'dialog',
  'hide_messages',
  'clear_history',
  'delete_messages'
])

type Status = 'idle' | 'ready' | 'scope' | 'down'

export const useChatStore = defineStore('chat', () => {
  const api = useChatApi()
  const model = reactive<ChatModel>({
    me: '',
    conversations: {},
    windows: {},
    users: {}
  })
  const state = ref<ChatState | null>(null)
  const status = ref<Status>('idle')
  const muted = ref(false)
  const folders = reactive<
    Record<ChatFolder, { next?: string; loaded: boolean; loading: boolean }>
  >({
    inbox: { loaded: false, loading: false },
    archive: { loaded: false, loading: false },
    requests: { loaded: false, loading: false }
  })
  const reactions = ref<KunChatReactionOption[]>([])
  const typing = reactive<Record<string, KunChatTypingEvent[]>>({})
  const openId = ref<string | null>(null)

  let seq = 0
  let syncing: Promise<void> | null = null
  let again = false
  let gapTimer: ReturnType<typeof setTimeout> | null = null
  let stateTimer: ReturnType<typeof setTimeout> | null = null
  const early = new Map<number, { u: ChatUpdate; x: ChatUpdateExtras }>()

  const users = computed(() => Object.values(model.users))

  const hasUnread = computed(
    () =>
      !muted.value &&
      !!state.value &&
      (state.value.unread_conversation_count > 0 ||
        state.value.request_count > 0)
  )

  const list = (folder: ChatFolder) =>
    Object.values(model.conversations)
      .filter(
        (c) =>
          chatFolderOf(c) === folder &&
          (c.last_message || c.id === openId.value)
      )
      .sort(compareChatConversations)

  const failed = (problem: ClientProblem) => {
    if (problem.code === 'SCOPE_REQUIRED') {
      status.value = 'scope'
    }
  }

  const putConversation = (c: WithUsers<ChatConversation>) => {
    mergeChatUsers(model, c.users)
    const { users: _users, ...rest } = c
    model.conversations[c.id] = rest as ChatConversation
  }

  const keepConversation = (c: ChatConversation) => {
    const prev = model.conversations[c.id]
    model.conversations[c.id] = c.pinned_messages
      ? c
      : {
          ...c,
          pinned_seqs: prev?.pinned_seqs,
          pinned_messages: prev?.pinned_messages
        }
  }

  const loadState = async () => {
    const r = await api.get<ChatState>('/state')
    if (r.ok) {
      state.value = r.data
    } else {
      failed(r.problem)
    }
    return r
  }

  const refreshState = () => {
    if (stateTimer) {
      return
    }
    stateTimer = setTimeout(() => {
      stateTimer = null
      void loadState()
    }, 800)
  }

  const refetch = async (id: string) => {
    const r = await api.get<ChatConversationDetail>(`/conversations/${id}`)
    if (r.ok) {
      putConversation(r.data)
    } else if (r.problem.status === 404) {
      Reflect.deleteProperty(model.conversations, id)
      Reflect.deleteProperty(model.windows, id)
    }
  }

  const page = async (id: string, params: Record<string, number>) => {
    const q = new URLSearchParams({ limit: String(PAGE) })
    for (const [k, v] of Object.entries(params)) {
      q.set(k, String(v))
    }
    const r = await api.get<ChatMessagePage>(
      `/conversations/${id}/messages?${q}`
    )
    if (!r.ok) {
      failed(r.problem)
      return null
    }
    mergeChatUsers(model, r.data.users)
    return r.data
  }

  // The update stream names message_reactions but carries no counts; only a
  // push does.
  const refreshReactions = async (id: string, target: number) => {
    const loaded = model.windows[id]?.items.some(
      (m) => !isPendingChatMessage(m) && m.seq === target
    )
    if (!loaded) {
      return
    }
    const p = await page(id, { around_seq: target, limit: 1 })
    const fresh = p?.items.find((m) => m.seq === target)
    const held = model.windows[id]?.items.find(
      (m) => !isPendingChatMessage(m) && m.seq === target
    )
    if (fresh && held) {
      held.reactions = fresh.reactions
    }
  }

  const apply = (u: ChatUpdate, x: ChatUpdateExtras, known?: Set<string>) => {
    const outcome = applyChatUpdate(model, u, x)
    if (
      outcome === 'refetch' &&
      (u.kind === 'pinned_messages' || !known?.has(u.conversation_id))
    ) {
      void refetch(u.conversation_id)
    }
    if (u.kind === 'message_reactions' && !x.reactions && !x.message) {
      void refreshReactions(u.conversation_id, Number(u.data.seq))
    }
    if (COUNTED.has(u.kind)) {
      refreshState()
    }
  }

  const loadFolder = async (folder: ChatFolder, more = false) => {
    const f = folders[folder]
    if (f.loading || (more && !f.next) || (!more && f.loaded)) {
      return
    }
    f.loading = true
    const q = new URLSearchParams({ folder, limit: '30' })
    if (more && f.next) {
      q.set('cursor', f.next)
    }
    const r = await api.get<ChatConversationPage>(`/conversations?${q}`)
    f.loading = false
    if (!r.ok) {
      failed(r.problem)
      return
    }
    mergeChatUsers(model, r.data.users)
    for (const c of r.data.items) {
      keepConversation(c)
    }
    f.next = r.data.next_cursor || undefined
    f.loaded = true
  }

  const resetFolders = () => {
    for (const f of Object.values(folders)) {
      Object.assign(f, { loaded: false, next: undefined })
    }
  }

  const reset = async () => {
    const r = await loadState()
    if (!r.ok) {
      return
    }
    seq = r.data.last_update_seq
    early.clear()
    const shown = (Object.keys(folders) as ChatFolder[]).filter(
      (f) => folders[f].loaded
    )
    model.conversations = {}
    model.windows = {}
    resetFolders()
    for (const f of shown) {
      await loadFolder(f)
    }
    if (openId.value) {
      await open(openId.value)
    }
  }

  const sync = (): Promise<void> => {
    if (syncing) {
      again = true
      return syncing
    }
    syncing = (async () => {
      do {
        again = false
        for (;;) {
          const r = await api.get<ChatUpdatesPage>(
            `/updates?after=${seq}&limit=100`
          )
          if (!r.ok) {
            failed(r.problem)
            return
          }
          if (r.data.too_long) {
            await reset()
            break
          }
          mergeChatUsers(model, r.data.users)
          const byKey = new Map(
            r.data.messages.map((m) => [`${m.conversation_id}:${m.seq}`, m])
          )
          const known = new Set(r.data.conversations.map((c) => c.id))
          for (const u of r.data.updates) {
            if (u.update_seq <= seq) {
              continue
            }
            apply(
              u,
              { message: byKey.get(`${u.conversation_id}:${u.data.seq}`) },
              known
            )
            seq = u.update_seq
          }
          for (const c of r.data.conversations) {
            if (model.conversations[c.id] || c.last_message) {
              keepConversation(c)
            }
          }
          if (!r.data.has_more) {
            break
          }
        }
      } while (again)
      for (const k of early.keys()) {
        if (k <= seq) {
          early.delete(k)
        }
      }
    })().finally(() => {
      syncing = null
    })
    return syncing
  }

  const drain = () => {
    for (let next = early.get(seq + 1); next; next = early.get(seq + 1)) {
      early.delete(seq + 1)
      apply(next.u, next.x)
      seq = next.u.update_seq
    }
    if (!early.size && gapTimer) {
      clearTimeout(gapTimer)
      gapTimer = null
    }
  }

  const receive = (u: ChatUpdate, x: ChatUpdateExtras) => {
    if (u.update_seq <= seq) {
      return
    }
    if (syncing) {
      again = true
      return
    }
    if (u.update_seq > seq + 1) {
      early.set(u.update_seq, { u, x })
      gapTimer ??= setTimeout(() => {
        gapTimer = null
        early.clear()
        void sync()
      }, GAP_WAIT)
      return
    }
    apply(u, x)
    seq = u.update_seq
    drain()
  }

  const push = (p: ChatPush) => {
    if (p.type === 'typing') {
      const now = Date.now()
      typing[p.conversation_id] = [
        ...(typing[p.conversation_id] ?? []).filter(
          (t) => t.user_id !== p.user_id && now - t.at < TYPING_TTL
        ),
        { user_id: p.user_id, at: now }
      ]
      return
    }
    if (p.update.kind === 'new_message' && p.message) {
      const sender = p.message.sender_id
      typing[p.update.conversation_id] = (
        typing[p.update.conversation_id] ?? []
      ).filter((t) => t.user_id !== sender)
    }
    receive(p.update, { message: p.message, reactions: p.reactions })
  }

  const loadMuted = async () => {
    const r = await settle(useApiClient().GET('/me/notification-preferences'))
    if (r.ok) {
      muted.value = r.data.muted_types.includes('chat')
    }
  }

  const start = async (me: string) => {
    if (model.me === me && status.value === 'ready') {
      return
    }
    model.me = me
    void loadMuted()
    const r = await loadState()
    if (!r.ok) {
      status.value = r.problem.code === 'SCOPE_REQUIRED' ? 'scope' : 'down'
      return
    }
    seq = r.data.last_update_seq
    status.value = 'ready'
  }

  let reactionsLoad: Promise<void> | null = null
  const loadReactions = () => {
    reactionsLoad ??= api.get<ChatReactionList>('/reactions').then((v) => {
      if (v.ok) {
        reactions.value = v.data.items
      } else {
        reactionsLoad = null
      }
    })
    return reactionsLoad
  }

  const stop = () => {
    model.me = ''
    model.conversations = {}
    model.windows = {}
    model.users = {}
    state.value = null
    status.value = 'idle'
    muted.value = false
    seq = 0
    early.clear()
    resetFolders()
  }

  const setWindow = (id: string, p: ChatMessagePage) => {
    const pending = model.windows[id]?.items.filter(isPendingChatMessage) ?? []
    model.windows[id] = {
      items: [...p.items, ...pending],
      hasOlder: p.has_more_before,
      hasNewer: p.has_more_after
    }
  }

  const open = async (id: string) => {
    openId.value = id
    if (!model.conversations[id]) {
      await refetch(id)
    } else {
      void refetch(id)
    }
    const c = model.conversations[id]
    if (!c || model.windows[id]) {
      return
    }
    const p = await page(
      id,
      c.me.unread_count > 0 ? { around_seq: c.me.last_read_seq + 1 } : {}
    )
    if (p) {
      setWindow(id, p)
    }
  }

  const loadOlder = async (id: string) => {
    const w = model.windows[id]
    const first = w?.items.find((m) => m.seq > 0)
    if (!w || !first) {
      return
    }
    const p = await page(id, { before_seq: first.seq })
    if (!p) {
      return
    }
    w.items = [...p.items, ...w.items]
    w.hasOlder = p.has_more_before
  }

  const loadNewer = async (id: string) => {
    const w = model.windows[id]
    const last = w?.items.filter((m) => m.seq > 0).at(-1)
    if (!w || !last) {
      return
    }
    const p = await page(id, { after_seq: last.seq })
    if (!p) {
      return
    }
    w.hasNewer = false
    for (const m of p.items) {
      upsertChatMessage(w, m)
    }
    w.hasNewer = p.has_more_after
  }

  const jump = async (id: string, target: number) => {
    const p = await page(id, { around_seq: target })
    if (p) {
      setWindow(id, p)
    }
  }

  const latest = async (id: string) => {
    const p = await page(id, {})
    if (p) {
      setWindow(id, p)
    }
  }

  return {
    model,
    state,
    status,
    muted,
    folders,
    reactions,
    typing,
    openId,
    users,
    hasUnread,
    list,
    failed,
    putConversation,
    refreshState,
    refetch,
    start,
    stop,
    sync,
    push,
    loadFolder,
    loadReactions,
    open,
    loadOlder,
    loadNewer,
    jump,
    latest
  }
})
