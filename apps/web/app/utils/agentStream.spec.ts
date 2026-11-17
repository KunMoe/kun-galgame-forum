import { describe, expect, it } from 'vitest'
import {
  agentPartsFromWire,
  applyAgentFrame,
  createSseParser,
  type SseEvent
} from './agentStream'

const card = {
  kind: 'work',
  id: '3032',
  title: '私の親愛なる妹',
  localized: {},
  release_date: '2025-05-22',
  content_rating: 'all_ages',
  content_limit: 'sfw',
  cover: { hash: 'cbaf', thumbhash: null, sexual: null }
}

const message = (): AgentMessage => ({
  id: '12',
  parent_id: '11',
  role: 'assistant',
  parts: [],
  status: 'streaming',
  run_id: 'r1',
  sibling_ids: ['12'],
  feedback: null
})

describe('createSseParser', () => {
  it('joins events split across chunks and keeps their ids', () => {
    const seen: SseEvent[] = []
    const push = createSseParser((e) => seen.push(e))
    push('id: 1\ndata: {"type":"st')
    push('art"}\n\nid: 2\r\ndata: {"type":"finish"}\r\n\r\ndata: [DONE]\n\n')
    expect(seen).toEqual([
      { id: '1', data: '{"type":"start"}' },
      { id: '2', data: '{"type":"finish"}' },
      { id: undefined, data: '[DONE]' }
    ])
  })
})

describe('applyAgentFrame', () => {
  it('builds text, reasoning, tool and card parts in stream order', () => {
    const m = message()
    const frames = [
      { type: 'start-step' },
      { type: 'reasoning-start', id: 'reasoning-0' },
      { type: 'reasoning-delta', id: 'reasoning-0', delta: '想想' },
      { type: 'reasoning-end', id: 'reasoning-0' },
      {
        type: 'tool-input-available',
        toolCallId: 'c1',
        toolName: 'search_catalog',
        input: { query: '妹妹' }
      },
      {
        type: 'tool-output-available',
        toolCallId: 'c1',
        output: {
          cards: 1,
          sources: 0,
          entities: [card, { id: '9' }],
          links: [{ url: 'https://example.com', title: 'Example' }, {}],
          total: 22,
          meta: { sort: 'classic', tags: ['催泪'] }
        }
      },
      { type: 'data-entity-card', id: 'work-3032', data: card },
      { type: 'data-entity-card', id: 'work-3032', data: card },
      { type: 'finish-step' },
      { type: 'start-step' },
      { type: 'text-start', id: 'text-1' },
      { type: 'text-delta', id: 'text-1', delta: '推荐' },
      { type: 'text-delta', id: 'text-1', delta: '这部' },
      { type: 'finish' }
    ]
    frames.forEach((f) => applyAgentFrame(m, f))
    expect(m.status).toBe('complete')
    expect(m.parts.map((p) => p.type)).toEqual([
      'step',
      'reasoning',
      'tool',
      'card',
      'step',
      'text'
    ])
    expect(m.parts[1]).toMatchObject({ text: '想想', done: true })
    expect(m.parts[2]).toMatchObject({
      name: 'search_catalog',
      input: { query: '妹妹' },
      output: {
        cards: 1,
        sources: 0,
        entities: [card],
        links: [{ url: 'https://example.com', title: 'Example' }],
        total: 22,
        meta: { sort: 'classic', tags: ['催泪'] }
      },
      done: true
    })
    expect(m.parts[5]).toMatchObject({ text: '推荐这部' })
  })

  it('keeps one browse link per filter', () => {
    const m = message()
    const data = {
      target: 'works',
      ids: ['118'],
      names: ['治愈'],
      sort: 'popular',
      sorts: ['popular'],
      total: 38
    }
    applyAgentFrame(m, { type: 'data-browse', id: 'browse-works:118', data })
    applyAgentFrame(m, { type: 'data-browse', id: 'browse-works:118', data })
    applyAgentFrame(m, { type: 'data-browse', id: 'bad', data: { ids: [] } })
    expect(m.parts).toEqual([
      { type: 'browse', id: 'browse-works:118', browse: data }
    ])
  })

  it('marks errors and aborts', () => {
    const failed = message()
    applyAgentFrame(failed, { type: 'error', errorText: 'boom' })
    applyAgentFrame(failed, { type: 'finish' })
    expect(failed).toMatchObject({ status: 'failed', error: 'boom' })

    const aborted = message()
    applyAgentFrame(aborted, { type: 'abort' })
    expect(aborted.status).toBe('canceled')
  })
})

describe('agentPartsFromWire', () => {
  it('reads stored steps, tool calls, cards, sources, text and files', () => {
    const parts = agentPartsFromWire([
      { type: 'file', mediaType: 'text/plain', filename: 'a.txt', text: 'x' },
      {
        type: 'file',
        uploadId: 'u1',
        mediaType: 'image/webp',
        filename: 'b.png',
        url: '/v2/agent/uploads/u1'
      },
      { type: 'step-start' },
      {
        type: 'tool-search_catalog',
        toolCallId: 'c1',
        state: 'output-available',
        input: { query: '妹妹' },
        output: { cards: 1, sources: 0 }
      },
      { type: 'data-entity-card', id: 'work-3032', data: card },
      { type: 'text', text: '你好' },
      {
        type: 'source-url',
        sourceId: 's1',
        url: 'https://example.com',
        title: 'Example'
      }
    ])
    expect(parts).toEqual([
      {
        type: 'file',
        mediaType: 'text/plain',
        filename: 'a.txt',
        text: 'x',
        uploadId: '',
        url: ''
      },
      {
        type: 'file',
        mediaType: 'image/webp',
        filename: 'b.png',
        text: '',
        uploadId: 'u1',
        url: '/v2/agent/uploads/u1'
      },
      { type: 'step', id: 'st2' },
      {
        type: 'tool',
        id: 'c1',
        name: 'search_catalog',
        input: { query: '妹妹' },
        output: {
          cards: 1,
          sources: 0,
          entities: [],
          links: [],
          total: null,
          meta: {}
        },
        done: true
      },
      { type: 'card', id: 'work-3032', card },
      { type: 'text', id: 't5', text: '你好' },
      { type: 'source', id: 's1', url: 'https://example.com', title: 'Example' }
    ])
  })
})

describe('failed tool calls', () => {
  it('marks a streamed tool-output-error as a failed, finished call', () => {
    const m = message()
    applyAgentFrame(m, {
      type: 'tool-input-available',
      toolCallId: 'c1',
      toolName: 'web_search',
      input: { query: '动画' }
    })
    applyAgentFrame(m, {
      type: 'tool-output-error',
      toolCallId: 'c1',
      errorText: '暂时无法使用'
    })
    expect(m.parts).toEqual([
      {
        type: 'tool',
        id: 'c1',
        name: 'web_search',
        input: { query: '动画' },
        output: null,
        done: true,
        error: '暂时无法使用'
      }
    ])
  })

  it('reads a stored output-error part back as failed', () => {
    const [part] = agentPartsFromWire([
      {
        type: 'tool-kungal__search_topics',
        toolCallId: 'c2',
        state: 'output-error',
        input: { query: '心跳' },
        errorText: '查询超时'
      }
    ])
    expect(part).toEqual({
      type: 'tool',
      id: 'c2',
      name: 'kungal__search_topics',
      input: { query: '心跳' },
      output: null,
      done: true,
      error: '查询超时'
    })
  })
})
