import { agentCardKey, agentCardKindLabel } from './agentCards'
import { agentMemoryChange, agentPastConversations } from './agentMemory'

type Part<T extends AgentPart['type']> = Extract<AgentPart, { type: T }>
export type AgentToolCall = Part<'tool'>
export type AgentNote = Part<'text'> | Part<'reasoning'>

export interface AgentStep {
  key: string
  notes: AgentNote[]
  calls: AgentToolCall[]
}

export interface AgentReplyLayout {
  steps: AgentStep[]
  answer: AgentNote[]
  cards: AgentEntityCard[]
  sources: Part<'source'>[]
  browses: AgentBrowse[]
}

const isNote = (p: AgentPart): p is AgentNote =>
  p.type === 'text' || p.type === 'reasoning'

const stepOf = (key: string, parts: AgentPart[]): AgentStep => ({
  key,
  notes: parts.filter((p): p is AgentNote => isNote(p) && !!p.text.trim()),
  calls: parts.filter((p): p is AgentToolCall => p.type === 'tool')
})

export const layoutReply = (parts: AgentPart[]): AgentReplyLayout => {
  const segments: AgentPart[][] = [[]]
  let stepped = false
  for (const p of parts) {
    if (p.type !== 'step') segments.at(-1)!.push(p)
    else if (stepped || segments[0]!.length) segments.push([])
    if (p.type === 'step') stepped = true
  }

  const cards = new Map<string, AgentEntityCard>()
  const sources = new Map<string, Part<'source'>>()
  const browses = new Map<string, AgentBrowse>()
  for (const p of parts) {
    if (p.type === 'card') cards.set(agentCardKey(p.card), p.card)
    if (p.type === 'source' && !sources.has(p.url)) sources.set(p.url, p)
    if (p.type === 'browse') browses.set(p.id, p.browse)
  }
  const extras = {
    cards: [...cards.values()],
    sources: [...sources.values()],
    browses: [...browses.values()]
  }

  if (!stepped) {
    const step = stepOf('s0', parts)
    return {
      steps: step.calls.length ? [{ ...step, notes: [] }] : [],
      answer: step.notes,
      ...extras
    }
  }

  const last = segments.at(-1)!
  const closed = last.some((p) => p.type === 'tool')
  const steps = (closed ? segments : segments.slice(0, -1))
    .map((s, i) => stepOf(`s${i}`, s))
    .filter((s) => s.calls.length || s.notes.length)
  return { steps, answer: closed ? [] : last.filter(isNote), ...extras }
}

const inputOf = (input: unknown): Record<string, unknown> => {
  if (typeof input === 'string') {
    try {
      return inputOf(JSON.parse(input))
    } catch {
      return {}
    }
  }
  return input && typeof input === 'object'
    ? (input as Record<string, unknown>)
    : {}
}

const quoted = (v: unknown) =>
  typeof v === 'string' && v.trim() ? `「${v.trim()}」` : ''

const namesOf = (v: unknown) =>
  Array.isArray(v)
    ? v.filter((n): n is string => typeof n === 'string' && !!n.trim())
    : []

const quotedList = (names: string[]) =>
  names.slice(0, 3).map(quoted).join('') +
  (names.length > 3 ? ` 等 ${names.length} 个` : '')

const SORTS: Record<string, string> = {
  popular: '热门',
  classic: '经典',
  newest: '最新'
}

const counted = (n: number, total: number | null, unit: string) =>
  `${n} ${unit}${total ? ` / 共 ${total} ${unit}` : ''}`

const hostOf = (v: unknown) => {
  try {
    return new URL(String(v)).hostname
  } catch {
    return ''
  }
}

export const describeToolCall = (
  call: AgentToolCall,
  titles: Map<string, string>
): string => {
  if (call.error) {
    const pending = { ...call, done: false, output: null, error: undefined }
    const what = describeToolCall(pending, titles).replace(/^正在/, '')
    return `${what}失败：${call.error}`
  }
  const input = inputOf(call.input)
  const state = call.done ? '已' : '正在'
  const out = call.output
  const tool = call.name.includes('__')
    ? `site:${call.name.split('__').at(-1)}`
    : call.name
  switch (tool) {
    case 'search_catalog': {
      const head = `${state}检索鲲 Galgame 论坛数据${quoted(input.query)}`
      if (!call.done || !out) return head
      return out.cards
        ? `${head} -> ${out.cards} 部`
        : `${head}，没有匹配的作品`
    }
    case 'get_work': {
      const id = String(input.id ?? '')
      const title = quoted(titles.get(agentCardKey({ kind: 'work', id })))
      return `${state}查看作品${title || (id ? ` #${id}` : '详情')}`
    }
    case 'search_entities': {
      const label = agentCardKindLabel(String(input.kind ?? '')) || '条目'
      const head = `${state}查找${label}${quoted(input.query)}`
      if (!call.done || !out) return head
      return out.cards ? `${head} -> ${out.cards} 个` : `${head}，没有找到`
    }
    case 'get_entity': {
      const kind = String(input.kind ?? '')
      const id = String(input.id ?? '')
      const title = quoted(titles.get(agentCardKey({ kind, id })))
      return `${state}查看${agentCardKindLabel(kind) || '条目'}${title || (id ? ` #${id}` : '详情')}`
    }
    case 'search_tags': {
      const forTraits =
        input.for === 'characters' || out?.meta.for === 'characters'
      const noun = forTraits ? '角色特征' : '标签'
      const head = `${state}查找${noun}${quotedList(namesOf(input.keywords))}`
      if (!call.done || !out) return head
      const matched = namesOf(out.meta.matched)
      if (!matched.length) return `${head}，没有匹配的${noun}`
      const more = matched.length > 4 ? ` 等 ${matched.length} 个` : ''
      return `${head} -> ${matched.slice(0, 4).join('、')}${more}`
    }
    case 'browse_works': {
      const sort = SORTS[String(out?.meta.sort ?? input.sort ?? '')] ?? ''
      const head = `${state}按标签${quotedList(namesOf(out?.meta.tags))}筛选${sort}作品`
      if (!call.done || !out) return head
      return out.cards
        ? `${head} -> ${counted(out.cards, out.total, '部')}`
        : `${head}，没有符合的作品`
    }
    case 'browse_characters': {
      const head = `${state}按特征${quotedList(namesOf(out?.meta.traits))}筛选角色`
      if (!call.done || !out) return head
      return out.cards
        ? `${head} -> ${counted(out.cards, out.total, '个')}`
        : `${head}，没有符合的角色`
    }
    case 'web_search': {
      const head = `${state}联网搜索${quoted(input.query)}`
      return call.done && out ? `${head} -> ${out.sources} 条结果` : head
    }
    case 'read_url': {
      const host = hostOf(input.url)
      return `${state}阅读网页${host ? ` ${host}` : ''}`
    }
    case 'remember': {
      if (!call.done) return `正在记住${quoted(input.content)}`
      const m = agentMemoryChange(out?.meta)
      if (!m) return '没能记住这件事'
      return `${m.op === 'update' ? '已更新记忆' : '已记住'}：${m.content}`
    }
    case 'forget': {
      if (!call.done) return '正在忘记一条记忆'
      const m = agentMemoryChange(out?.meta)
      return m ? `已忘记：${m.content}` : '没能忘记这条记忆'
    }
    case 'search_conversations': {
      const head = `${state}搜索过去的对话${quoted(input.query)}`
      if (!call.done || !out) return head
      const n = out.total ?? agentPastConversations(out.meta).length
      return n ? `${head} -> ${n} 个` : `${head}，没有找到`
    }
    case 'read_conversation': {
      const title = agentPastConversations(out?.meta)[0]?.title
      return `${state}阅读过去的对话${quoted(title)}`
    }
    case 'site:search_topics': {
      const head = `${state}搜索论坛话题${quoted(input.query)}`
      if (!call.done || !out) return head
      return out.cards ? `${head} -> ${out.cards} 个` : `${head}，没有找到`
    }
    case 'site:get_topic': {
      const id = String(input.topic_id ?? '')
      const title = quoted(
        titles.get(agentCardKey({ kind: 'kungal:topic', id }))
      )
      return `${state}阅读话题${title || (id ? ` #${id}` : '')}`
    }
    default:
      return `${state}查资料`
  }
}
