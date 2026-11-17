import { agentProblemMessage } from '#shared/utils/api/agent'
import { settle } from '#shared/utils/api/problem'
import { agentCardKey } from './agentCards'

type Frame = Record<string, unknown>

const str = (v: unknown) => (typeof v === 'string' ? v : '')

const TOOL_FAILED = '暂时无法使用'
const errorOf = (v: unknown) => str(v).trim() || TOOL_FAILED

const cardOf = (data: unknown): AgentEntityCard | null =>
  data && typeof data === 'object' && 'kind' in data
    ? (data as AgentEntityCard)
    : null

const linkOf = (v: unknown): AgentLink[] => {
  const l = v && typeof v === 'object' ? (v as Record<string, unknown>) : {}
  return str(l.url) ? [{ url: str(l.url), title: str(l.title) }] : []
}

const listOf = (v: unknown) => (Array.isArray(v) ? (v as unknown[]) : [])

const strings = (v: unknown) =>
  listOf(v).filter((x): x is string => typeof x === 'string')

const browseOf = (data: unknown): AgentBrowse | null => {
  if (!data || typeof data !== 'object') return null
  const d = data as Record<string, unknown>
  const ids = strings(d.ids)
  if (!str(d.target) || !ids.length) return null
  return {
    target: str(d.target),
    ids,
    names: strings(d.names),
    sort: str(d.sort),
    sorts: strings(d.sorts),
    total: typeof d.total === 'number' ? d.total : 0
  }
}

const toolOutputOf = (v: unknown): AgentToolOutput | null => {
  if (!v || typeof v !== 'object') return null
  const o = v as Record<string, unknown>
  return {
    cards: typeof o.cards === 'number' ? o.cards : 0,
    sources: typeof o.sources === 'number' ? o.sources : 0,
    entities: listOf(o.entities).flatMap((e) => cardOf(e) ?? []),
    links: listOf(o.links).flatMap(linkOf),
    total: typeof o.total === 'number' ? o.total : null,
    meta:
      o.meta && typeof o.meta === 'object'
        ? (o.meta as Record<string, unknown>)
        : {}
  }
}

export const agentPartsFromWire = (
  parts: Record<string, unknown>[] | null
): AgentPart[] => {
  const out: AgentPart[] = []
  ;(parts ?? []).forEach((p, i) => {
    const type = str(p.type)
    if (type.startsWith('tool-')) {
      out.push({
        type: 'tool',
        id: str(p.toolCallId) || `c${i}`,
        name: type.slice(5),
        input: p.input,
        output: toolOutputOf(p.output),
        done: true,
        ...(p.state === 'output-error' && { error: errorOf(p.errorText) })
      })
      return
    }
    switch (type) {
      case 'step-start':
        out.push({ type: 'step', id: `st${i}` })
        break
      case 'text':
        out.push({ type: 'text', id: `t${i}`, text: str(p.text) })
        break
      case 'reasoning':
        out.push({
          type: 'reasoning',
          id: `r${i}`,
          text: str(p.text),
          done: true
        })
        break
      case 'file':
        out.push({
          type: 'file',
          filename: str(p.filename),
          mediaType: str(p.mediaType),
          text: str(p.text),
          uploadId: str(p.uploadId),
          url: str(p.url)
        })
        break
      case 'source-url':
        out.push({
          type: 'source',
          id: str(p.sourceId) || `s${i}`,
          url: str(p.url),
          title: str(p.title)
        })
        break
      case 'data-entity-card': {
        const card = cardOf(p.data)
        if (card) out.push({ type: 'card', id: agentCardKey(card), card })
        break
      }
      case 'data-browse': {
        const browse = browseOf(p.data)
        if (browse)
          out.push({ type: 'browse', id: str(p.id) || `b${i}`, browse })
        break
      }
    }
  })
  return out
}

const findPart = <T extends AgentPart['type']>(
  m: AgentMessage,
  type: T,
  id: string
) =>
  m.parts.find((p) => p.type === type && 'id' in p && p.id === id) as
    | Extract<AgentPart, { type: T }>
    | undefined

const appendText = (
  m: AgentMessage,
  type: 'text' | 'reasoning',
  id: string,
  delta: string
) => {
  const part = findPart(m, type, id)
  if (part) {
    part.text += delta
    return
  }
  m.parts.push(
    type === 'text'
      ? { type, id, text: delta }
      : { type, id, text: delta, done: false }
  )
}

export const applyAgentFrame = (m: AgentMessage, f: Frame) => {
  const id = str(f.id)
  switch (f.type) {
    case 'start-step':
      m.parts.push({ type: 'step', id: `st${m.parts.length}` })
      break
    case 'text-start':
    case 'reasoning-start':
      appendText(m, f.type === 'text-start' ? 'text' : 'reasoning', id, '')
      break
    case 'text-delta':
    case 'reasoning-delta':
      appendText(
        m,
        f.type === 'text-delta' ? 'text' : 'reasoning',
        id,
        str(f.delta)
      )
      break
    case 'reasoning-end': {
      const part = findPart(m, 'reasoning', id)
      if (part) part.done = true
      break
    }
    case 'tool-input-start':
    case 'tool-input-available': {
      const callId = str(f.toolCallId)
      const part = findPart(m, 'tool', callId)
      if (part) part.input = f.input ?? part.input
      else
        m.parts.push({
          type: 'tool',
          id: callId,
          name: str(f.toolName),
          input: f.input,
          output: null,
          done: false
        })
      break
    }
    case 'tool-output-available': {
      const part = findPart(m, 'tool', str(f.toolCallId))
      if (part) {
        part.output = toolOutputOf(f.output)
        part.done = true
      }
      break
    }
    case 'tool-output-error': {
      const part = findPart(m, 'tool', str(f.toolCallId))
      if (part) {
        part.output = null
        part.done = true
        part.error = errorOf(f.errorText)
      }
      break
    }
    case 'source-url':
      m.parts.push({
        type: 'source',
        id: str(f.sourceId),
        url: str(f.url),
        title: str(f.title)
      })
      break
    case 'data-entity-card': {
      const card = cardOf(f.data)
      if (card && !findPart(m, 'card', id))
        m.parts.push({ type: 'card', id, card })
      break
    }
    case 'data-browse': {
      const browse = browseOf(f.data)
      if (browse && !findPart(m, 'browse', id))
        m.parts.push({ type: 'browse', id, browse })
      break
    }
    case 'error':
      m.status = 'failed'
      m.error = str(f.errorText)
      break
    case 'abort':
      m.status = 'canceled'
      break
    case 'finish':
      if (m.status === 'streaming') m.status = 'complete'
      break
  }
}

export interface SseEvent {
  id?: string
  data: string
}

export const createSseParser = (onEvent: (e: SseEvent) => void) => {
  let buffer = ''
  let id: string | undefined
  let data: string[] = []
  return (chunk: string) => {
    buffer += chunk
    let nl: number
    while ((nl = buffer.indexOf('\n')) >= 0) {
      const line = buffer.slice(0, nl).replace(/\r$/, '')
      buffer = buffer.slice(nl + 1)
      if (line === '') {
        if (data.length) onEvent({ id, data: data.join('\n') })
        id = undefined
        data = []
      } else if (line.startsWith('data:')) {
        data.push(line.slice(5).replace(/^ /, ''))
      } else if (line.startsWith('id:')) {
        id = line.slice(3).trim()
      }
    }
  }
}

export const readAgentStream = async (
  url: string,
  lastEventId: string | undefined,
  signal: AbortSignal,
  onEvent: (id: string | undefined, data: string) => void
) => {
  const res = await fetch(url, {
    credentials: 'include',
    headers: lastEventId ? { 'Last-Event-ID': lastEventId } : {},
    signal
  })
  if (res.status === 204) return 'gone' as const
  if (!res.ok || !res.body) {
    const error = await res.json().catch(() => undefined)
    const r = await settle(Promise.resolve({ error, response: res }))
    throw new Error(r.ok ? 'AI 助手暂不可用' : agentProblemMessage(r.problem))
  }
  const push = createSseParser((e) => onEvent(e.id, e.data))
  const reader = res.body.pipeThrough(new TextDecoderStream()).getReader()
  for (;;) {
    const { value, done } = await reader.read()
    if (done) return 'done' as const
    push(value)
  }
}
