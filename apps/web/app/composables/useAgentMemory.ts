import { agentProblemMessage } from '#shared/utils/api/agent'
import type { ClientProblem } from '#shared/utils/api/problem'
type Switches = Partial<Pick<AgentPreferences, 'memory' | 'history'>>

export const useAgentMemory = () => {
  const api = useAgentApi()
  const state = useState('kun-agent-memory', () => ({
    isOpen: false,
    loading: false,
    prefs: null as AgentPreferences | null,
    items: [] as AgentMemory[],
    max: 0
  }))

  const fail = (problem: ClientProblem) => {
    useMessage(agentProblemMessage(problem), 'warn')
    return false
  }

  const loadItems = async () => {
    const r = await api.get<AgentMemoryList>('/memories')
    if (!r.ok) return fail(r.problem)
    state.value.items = r.data.items ?? []
    state.value.max = r.data.max
    return true
  }

  const load = async () => {
    state.value.loading = true
    const [p] = await Promise.all([
      api.get<AgentPreferences>('/preferences'),
      loadItems()
    ])
    state.value.loading = false
    if (!p.ok) return fail(p.problem)
    state.value.prefs = p.data
  }

  const open = () => {
    state.value.isOpen = true
    void load()
  }

  const close = () => {
    state.value.isOpen = false
  }

  const setPrefs = async (change: Switches) => {
    const prev = state.value.prefs
    if (prev) state.value.prefs = { ...prev, ...change }
    const r = await api.patch<AgentPreferences>('/preferences', change)
    if (!r.ok) {
      state.value.prefs = prev
      return fail(r.problem)
    }
    state.value.prefs = r.data
    return true
  }

  const add = async (content: string) => {
    const r = await api.post<AgentMemory>('/memories', { content })
    if (!r.ok) return fail(r.problem)
    await loadItems()
    return true
  }

  const update = async (id: string, content: string) => {
    const r = await api.patch<AgentMemory>(`/memories/${id}`, {
      content
    })
    if (!r.ok) return fail(r.problem)
    state.value.items = state.value.items.map((m) => (m.id === id ? r.data : m))
    return true
  }

  const remove = async (id: string) => {
    const r = await api.del(`/memories/${id}`)
    if (!r.ok) return fail(r.problem)
    state.value.items = state.value.items.filter((m) => m.id !== id)
    return true
  }

  const clear = async () => {
    const r = await api.del('/memories')
    if (!r.ok) return fail(r.problem)
    state.value.items = []
    return true
  }

  return { state, open, close, setPrefs, add, update, remove, clear }
}
