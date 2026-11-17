export const graftTurn = (
  thread: AgentMessage[],
  anchorId: string | null,
  added: AgentMessage[]
) => {
  const cut = anchorId ? thread.findIndex((m) => m.id === anchorId) : -1
  const first = added[0]
  if (!first) return
  const replaced = thread[cut + 1]
  first.sibling_ids = replaced
    ? [...replaced.sibling_ids, first.id]
    : [first.id]
  thread.splice(cut + 1, thread.length, ...added)
}

export const messageText = (m: AgentMessage) =>
  m.parts
    .flatMap((p) => (p.type === 'text' && p.text ? [p.text] : []))
    .join('\n\n')
