import type { AgentApi } from '#shared/utils/api/agent'

const PAGE = 30
const STARRED_LIMIT = 100

const newestFirst = (a: AgentConversation, b: AgentConversation) =>
  b.updated_at.localeCompare(a.updated_at)

export const useAgentConversations = (api: AgentApi) => {
  const recent = ref<AgentConversation[]>([])
  const starred = ref<AgentConversation[]>([])
  const cursor = ref<string | null>(null)
  const loadingMore = ref(false)

  const listOf = (c: AgentConversation) => (c.starred ? starred : recent)

  const find = (id: string) =>
    starred.value.find((c) => c.id === id) ??
    recent.value.find((c) => c.id === id)

  const fetchStarred = async () => {
    const r = await api.get<AgentConversationPage>(
      `/conversations?starred=true&limit=${STARRED_LIMIT}`
    )
    if (r.ok) starred.value = r.data.items ?? []
  }

  const fetchRecent = async (limit: number) => {
    const r = await api.get<AgentConversationPage>(
      `/conversations?starred=false&limit=${limit}`
    )
    if (!r.ok) return
    recent.value = r.data.items ?? []
    cursor.value = r.data.next_cursor
  }

  const refresh = () =>
    Promise.all([
      fetchStarred(),
      fetchRecent(Math.min(100, Math.max(PAGE, recent.value.length)))
    ])

  const loadMore = async () => {
    if (!cursor.value || loadingMore.value) return
    loadingMore.value = true
    const r = await api.get<AgentConversationPage>(
      `/conversations?starred=false&limit=${PAGE}&cursor=${encodeURIComponent(cursor.value)}`
    )
    loadingMore.value = false
    if (!r.ok) return
    const known = new Set(recent.value.map((c) => c.id))
    recent.value.push(...(r.data.items ?? []).filter((c) => !known.has(c.id)))
    cursor.value = r.data.next_cursor
  }

  const drop = (id: string) => {
    recent.value = recent.value.filter((c) => c.id !== id)
    starred.value = starred.value.filter((c) => c.id !== id)
  }

  const put = (c: AgentConversation) => {
    drop(c.id)
    const list = listOf(c)
    list.value = [...list.value, c].sort(newestFirst)
  }

  const patch = (id: string, change: Partial<AgentConversation>) => {
    const c = find(id)
    if (c) put({ ...c, ...change })
  }

  return {
    recent,
    starred,
    cursor,
    loadingMore,
    find,
    refresh,
    loadMore,
    drop,
    put,
    patch
  }
}
