import { defineStore } from 'pinia'
import {
  agentPath,
  agentProblemMessage,
  isAgentDisabled,
  isAgentQuotaProblem
} from '#shared/utils/api/agent'
import type { ApiResult, ClientProblem } from '#shared/utils/api/problem'
import {
  agentPartsFromWire,
  applyAgentFrame,
  readAgentStream
} from '~/utils/agentStream'
import { graftTurn } from '~/utils/agentThread'
import { agentFileWire, type AgentFilePart } from '~/utils/agentUpload'
import { useAgentConversations } from './agentConversations'

interface SendOptions {
  parentId?: string
  webSearch?: boolean
}

const RECONNECTS = 3

const fromWire = (w: AgentWireMessage): AgentMessage => ({
  id: w.id,
  parent_id: w.parent_id,
  role: w.role,
  parts: agentPartsFromWire(w.parts),
  status: w.status,
  run_id: w.run_id,
  sibling_ids: w.sibling_ids?.length ? w.sibling_ids : [w.id],
  feedback: w.feedback ?? null
})

export const useAgentStore = defineStore('kungal-agent', () => {
  const api = useAgentApi()
  const origin = useRuntimeConfig().public.apiBaseUrl as string
  const list = useAgentConversations(api)

  const persona = ref<AgentPersona | null>(null)
  const quota = ref<AgentQuota | null>(null)
  const status = ref<'idle' | 'ready' | 'down' | 'disabled'>('idle')
  const threads = reactive(new Map<string, AgentMessage[]>())
  const running = reactive(new Map<string, string>())
  const webSearch = ref(false)

  const controllers = new Map<string, AbortController>()

  const webSearchAvailable = computed(
    () => !!persona.value?.web_search_available
  )

  const warn = (problem: ClientProblem) => {
    if (isAgentDisabled(problem)) status.value = 'disabled'
    useMessage(agentProblemMessage(problem), 'warn')
    if (isAgentQuotaProblem(problem)) void refreshQuota()
  }

  const init = async () => {
    if (status.value === 'ready') return
    const [p, q] = await Promise.all([
      api.get<AgentPersona>('/persona'),
      api.get<AgentQuota>('/quota'),
      list.refresh()
    ])
    if (!p.ok) {
      status.value = isAgentDisabled(p.problem) ? 'disabled' : 'down'
      return
    }
    persona.value = p.data
    if (q.ok) quota.value = q.data
    status.value = 'ready'
  }

  const refreshQuota = async () => {
    const q = await api.get<AgentQuota>('/quota')
    if (q.ok) quota.value = q.data
  }

  const threadOf = (id: string) => {
    if (!threads.has(id)) threads.set(id, [])
    return threads.get(id)!
  }

  const open = async (
    id: string,
    head?: string
  ): Promise<ApiResult<AgentConversationDetail> | undefined> => {
    if (running.has(id)) return
    const r = await api.get<AgentConversationDetail>(
      `/conversations/${id}${head ? `?head=${encodeURIComponent(head)}` : ''}`
    )
    if (!r.ok) return r
    threads.set(id, (r.data.messages ?? []).map(fromWire))
    const last = threads.get(id)!.at(-1)
    const live = r.data.active_run_id
    if (live && last?.role === 'assistant' && last.run_id === live) {
      last.parts = []
      last.status = 'streaming'
      void stream(id, live, last)
    }
    return r
  }

  const stream = async (convId: string, runId: string, msg: AgentMessage) => {
    const ctrl = new AbortController()
    controllers.set(convId, ctrl)
    running.set(convId, runId)
    const url = agentPath(origin, `/runs/${runId}/stream`)
    let lastId: string | undefined
    let finished = false
    const onEvent = (id: string | undefined, data: string) => {
      if (id) lastId = id
      if (data === '[DONE]') {
        finished = true
        return
      }
      const frame = JSON.parse(data) as Record<string, unknown>
      const d = frame.data as Record<string, unknown> | undefined
      if (frame.type === 'data-quota' && quota.value) {
        if (typeof d?.free_left === 'number')
          quota.value.free_left = d.free_left
      } else if (frame.type === 'data-title' && typeof d?.title === 'string') {
        list.patch(convId, { title: d.title })
      }
      applyAgentFrame(msg, frame)
    }
    let gone = false
    for (let attempt = 0; !finished && attempt <= RECONNECTS; attempt++) {
      try {
        if (
          (await readAgentStream(url, lastId, ctrl.signal, onEvent)) === 'gone'
        ) {
          gone = true
          break
        }
      } catch (err) {
        if (ctrl.signal.aborted) break
        if (attempt === RECONNECTS) {
          msg.status = 'failed'
          msg.error = (err as Error).message
        } else {
          await new Promise((r) => setTimeout(r, 1000 * (attempt + 1)))
        }
      }
    }
    if (msg.status === 'streaming' && !gone) msg.status = 'complete'
    controllers.delete(convId)
    running.delete(convId)
    if (gone) await open(convId)
    await list.refresh()
    if (!list.find(convId)?.title) setTimeout(() => void list.refresh(), 4000)
  }

  const begin = (
    convId: string,
    anchorId: string | null,
    r: AgentSendResult,
    user: AgentMessage | null
  ) => {
    const assistant: AgentMessage = {
      id: r.assistant_message_id,
      parent_id: r.user_message_id,
      role: 'assistant',
      parts: [],
      status: 'streaming',
      run_id: r.run_id,
      sibling_ids: [r.assistant_message_id],
      feedback: null
    }
    const thread = threadOf(convId)
    graftTurn(thread, anchorId, user ? [user, assistant] : [assistant])
    list.patch(convId, {
      updated_at: new Date().toISOString(),
      active_run_id: r.run_id
    })
    void stream(convId, r.run_id, thread.at(-1)!)
  }

  const postRun = async (path: string, body: object) => {
    const key = crypto.randomUUID()
    const post = () =>
      api.post<AgentSendResult>(path, body, { 'Idempotency-Key': key })
    const r = await post()
    return !r.ok && r.problem.kind === 'network' ? post() : r
  }

  const send = async (
    convId: string,
    text: string,
    files: AgentFilePart[],
    opts: SendOptions = {}
  ) => {
    const parentId = opts.parentId ?? threads.get(convId)?.at(-1)?.id
    const textParts = text ? [{ type: 'text', text }] : []
    const r = await postRun(`/conversations/${convId}/messages`, {
      parts: [...textParts, ...files.map(agentFileWire)],
      ...(parentId ? { parent_id: parentId } : {}),
      web_search:
        webSearchAvailable.value && (opts.webSearch ?? webSearch.value)
    })
    if (!r.ok) {
      warn(r.problem)
      return false
    }
    begin(convId, parentId ?? null, r.data, {
      id: r.data.user_message_id,
      parent_id: parentId && parentId !== '0' ? parentId : null,
      role: 'user',
      parts: [...agentPartsFromWire(textParts), ...files],
      status: 'complete',
      run_id: r.data.run_id,
      sibling_ids: [r.data.user_message_id],
      feedback: null
    })
    return true
  }

  const regenerate = async (convId: string, userMessageId: string) => {
    if (running.has(convId)) return false
    const r = await postRun(`/conversations/${convId}/regenerate`, {
      message_id: userMessageId,
      web_search: webSearchAvailable.value && webSearch.value
    })
    if (!r.ok) {
      warn(r.problem)
      return false
    }
    begin(convId, userMessageId, r.data, null)
    return true
  }

  const switchTo = async (convId: string, messageId: string) => {
    if (running.has(convId)) return
    const r = await open(convId, messageId)
    if (!r) return
    if (!r.ok) {
      warn(r.problem)
      return
    }
    void api.patch(`/conversations/${convId}`, {
      head_message_id: messageId
    })
  }

  const feedback = async (msg: AgentMessage, rating: AgentFeedback) => {
    const prev = msg.feedback
    const next = prev === rating ? null : rating
    msg.feedback = next
    const r = await api.put(`/messages/${msg.id}/feedback`, {
      rating: next ?? 'none'
    })
    if (!r.ok) {
      msg.feedback = prev
      warn(r.problem)
    }
  }

  const cancel = async (convId: string) => {
    const runId = running.get(convId)
    if (!runId) return
    const r = await api.post(`/runs/${runId}/cancel`)
    if (!r.ok) warn(r.problem)
  }

  const create = async () => {
    const r = await api.post<AgentConversation>('/conversations', {
      mode: 'assistant'
    })
    if (!r.ok) {
      warn(r.problem)
      return null
    }
    list.put(r.data)
    threads.set(r.data.id, [])
    return r.data.id
  }

  const update = async (
    id: string,
    change: { title?: string; starred?: boolean }
  ) => {
    const r = await api.patch<AgentConversation>(`/conversations/${id}`, change)
    if (!r.ok) {
      warn(r.problem)
      return false
    }
    list.put(r.data)
    return true
  }

  const remove = async (id: string) => {
    const r = await api.del(`/conversations/${id}`)
    if (!r.ok) {
      warn(r.problem)
      return false
    }
    controllers.get(id)?.abort()
    list.drop(id)
    threads.delete(id)
    return true
  }

  return {
    persona,
    quota,
    status,
    conversations: list.recent,
    starred: list.starred,
    cursor: list.cursor,
    loadingMore: list.loadingMore,
    findConversation: list.find,
    threads,
    running,
    webSearch,
    webSearchAvailable,
    init,
    loadMore: list.loadMore,
    refreshQuota,
    open,
    create,
    send,
    regenerate,
    switchTo,
    feedback,
    cancel,
    update,
    remove
  }
})
