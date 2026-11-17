import { refDebounced } from '@vueuse/core'

const PAGE = 30
const MAX_QUERY = 100

export const useAgentConversationSearch = () => {
  const api = useAgentApi()
  const agent = useAgentStore()

  const term = ref('')
  const query = computed(() => term.value.trim().slice(0, MAX_QUERY))
  const settled = refDebounced(query, 300)
  const items = ref<AgentConversation[]>([])
  const cursor = ref<string | null>(null)
  const loading = ref(false)
  const loadingMore = ref(false)
  let seq = 0

  const urlOf = (q: string, after: string | null) =>
    `/conversations?q=${encodeURIComponent(q)}&limit=${PAGE}${
      after ? `&cursor=${encodeURIComponent(after)}` : ''
    }`

  const search = async (q: string) => {
    const mine = ++seq
    if (!q) {
      items.value = []
      cursor.value = null
      loading.value = false
      return
    }
    loading.value = true
    const r = await api.get<AgentConversationPage>(urlOf(q, null))
    if (mine !== seq) return
    loading.value = false
    if (!r.ok) return
    items.value = r.data.items ?? []
    cursor.value = r.data.next_cursor
  }

  const loadMore = async () => {
    if (!cursor.value || loadingMore.value) return
    const mine = seq
    loadingMore.value = true
    const r = await api.get<AgentConversationPage>(
      urlOf(settled.value, cursor.value)
    )
    loadingMore.value = false
    if (mine !== seq || !r.ok) return
    const known = new Set(items.value.map((c) => c.id))
    items.value.push(...(r.data.items ?? []).filter((c) => !known.has(c.id)))
    cursor.value = r.data.next_cursor
  }

  watch(settled, (q) => void search(q))
  watch(
    () => [agent.conversations, agent.starred],
    () => {
      if (settled.value) void search(settled.value)
    }
  )

  return {
    term,
    active: computed(() => !!query.value),
    pending: computed(() => loading.value || query.value !== settled.value),
    items,
    cursor,
    loadingMore,
    loadMore
  }
}
