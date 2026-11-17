export const AGENT_MEMORY_MAX_CHARS = 200

export interface AgentMemoryChange {
  op: 'add' | 'update' | 'delete'
  content: string
}

export interface AgentPastConversation {
  id: string
  title: string
  updated_at: string
}

const OPS = new Set(['add', 'update', 'delete'])

const recordOf = (v: unknown): Record<string, unknown> | null =>
  v && typeof v === 'object' && !Array.isArray(v)
    ? (v as Record<string, unknown>)
    : null

export const agentMemoryChange = (
  meta: Record<string, unknown> | undefined
): AgentMemoryChange | null => {
  const m = recordOf(meta?.memory)
  if (!m || typeof m.op !== 'string' || !OPS.has(m.op)) return null
  const content = typeof m.content === 'string' ? m.content.trim() : ''
  return content ? { op: m.op as AgentMemoryChange['op'], content } : null
}

export const agentPastConversations = (
  meta: Record<string, unknown> | undefined
): AgentPastConversation[] =>
  Array.isArray(meta?.conversations)
    ? meta.conversations.flatMap((v) => {
        const c = recordOf(v)
        const id = c?.id
        if (typeof id !== 'string' || !/^[1-9]\d*$/.test(id)) return []
        const title = typeof c?.title === 'string' ? c.title.trim() : ''
        const updated = c?.updated_at
        return [
          {
            id,
            title: title || '新对话',
            updated_at: typeof updated === 'string' ? updated : ''
          }
        ]
      })
    : []
