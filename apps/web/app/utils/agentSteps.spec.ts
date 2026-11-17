import { describe, expect, it } from 'vitest'
import { describeToolCall, layoutReply, type AgentToolCall } from './agentSteps'
import { graftTurn } from './agentThread'

const tool = (id: string, done = true): AgentToolCall => ({
  type: 'tool',
  id,
  name: 'search_catalog',
  input: { query: '猫耳 妹妹' },
  output: done
    ? { cards: 3, sources: 0, entities: [], links: [], total: null, meta: {} }
    : null,
  done
})

const msg = (id: string, sibling_ids: string[] = [id]): AgentMessage => ({
  id,
  parent_id: null,
  role: 'user',
  parts: [],
  status: 'complete',
  run_id: 'r',
  sibling_ids,
  feedback: null
})

describe('layoutReply', () => {
  it('folds tool steps away from the final answer', () => {
    const l = layoutReply([
      { type: 'step', id: 'a' },
      { type: 'text', id: 't0', text: '先查一下' },
      tool('c1'),
      { type: 'step', id: 'b' },
      tool('c2'),
      { type: 'step', id: 'c' },
      { type: 'text', id: 't2', text: '推荐这部' }
    ])
    expect(l.steps.map((s) => s.calls.map((c) => c.id))).toEqual([
      ['c1'],
      ['c2']
    ])
    expect(l.steps[0]!.notes).toMatchObject([{ text: '先查一下' }])
    expect(l.answer).toMatchObject([{ text: '推荐这部' }])
  })

  it('keeps an empty final step open for the answer that is about to stream', () => {
    const l = layoutReply([
      { type: 'step', id: 'a' },
      tool('c1'),
      { type: 'step', id: 'b' }
    ])
    expect(l.steps).toHaveLength(1)
    expect(l.answer).toEqual([])
  })

  it('treats a reply that stopped inside a tool step as all process', () => {
    const l = layoutReply([
      { type: 'step', id: 'a' },
      { type: 'text', id: 't0', text: '我去查' },
      tool('c1', false)
    ])
    expect(l.steps).toHaveLength(1)
    expect(l.answer).toEqual([])
  })

  it('reads a reply stored before steps were recorded', () => {
    const l = layoutReply([tool('c1'), { type: 'text', id: 't', text: '答' }])
    expect(l.steps[0]!.calls).toHaveLength(1)
    expect(l.steps[0]!.notes).toEqual([])
    expect(l.answer).toMatchObject([{ text: '答' }])
  })
})

describe('describeToolCall', () => {
  const withMeta = (
    name: string,
    input: unknown,
    meta: Record<string, unknown>,
    total: number | null = null
  ): AgentToolCall => ({
    ...tool('m'),
    name,
    input,
    output: { cards: 0, sources: 0, entities: [], links: [], total, meta }
  })

  it('says what it remembered or forgot', () => {
    const memory = (op: string) => ({
      memory: { op, id: '12', content: '用户喜欢纯爱' }
    })
    expect(
      describeToolCall(withMeta('remember', {}, memory('add')), new Map())
    ).toBe('已记住：用户喜欢纯爱')
    expect(
      describeToolCall(withMeta('remember', {}, memory('update')), new Map())
    ).toBe('已更新记忆：用户喜欢纯爱')
    expect(
      describeToolCall(withMeta('forget', {}, memory('delete')), new Map())
    ).toBe('已忘记：用户喜欢纯爱')
    expect(
      describeToolCall(
        { ...tool('m', false), name: 'remember', input: { content: '喜欢猫' } },
        new Map()
      )
    ).toBe('正在记住「喜欢猫」')
    expect(
      describeToolCall(
        { ...tool('m'), name: 'remember', output: null },
        new Map()
      )
    ).toBe('没能记住这件事')
    expect(
      describeToolCall(withMeta('forget', {}, { memory: 'x' }), new Map())
    ).toBe('没能忘记这条记忆')
  })

  it('says what it looked up among the forum topics', () => {
    expect(
      describeToolCall(
        withMeta('kungal__search_topics', { query: '千恋 攻略' }, {}),
        new Map()
      )
    ).toBe('已搜索论坛话题「千恋 攻略」，没有找到')
    expect(
      describeToolCall(
        {
          ...withMeta('kungal__search_topics', { query: '千恋' }, {}),
          output: {
            cards: 6,
            sources: 0,
            entities: [],
            links: [],
            total: null,
            meta: {}
          }
        },
        new Map()
      )
    ).toBe('已搜索论坛话题「千恋」 -> 6 个')
    expect(
      describeToolCall(
        withMeta('kungal__get_topic', { topic_id: 88 }, {}),
        new Map([['kungal:topic-88', '猫耳作品讨论']])
      )
    ).toBe('已阅读话题「猫耳作品讨论」')
    expect(
      describeToolCall(
        withMeta('kungal__get_topic', { topic_id: 9 }, {}),
        new Map()
      )
    ).toBe('已阅读话题 #9')
  })

  it('says what it looked up among the forum works, engines and tools', () => {
    const found = (name: string, input: object, cards: number) => ({
      ...withMeta(name, input, {}),
      output: {
        cards,
        sources: 0,
        entities: [],
        links: [],
        total: null,
        meta: {}
      }
    })
    expect(
      describeToolCall(
        found('kungal__search_galgames', { q: '千恋' }, 6),
        new Map()
      )
    ).toBe('已搜索本站作品「千恋」 -> 6 部')
    expect(
      describeToolCall(
        found('kungal__search_galgames', { mobile: true }, 0),
        new Map()
      )
    ).toBe('已搜索本站作品（按条件筛选），没有找到')
    expect(
      describeToolCall(
        withMeta('kungal__get_galgame', { id: 15 }, {}),
        new Map([['kungal:galgame-15', '千恋＊万花']])
      )
    ).toBe('已查看本站作品「千恋＊万花」的资源与评价')
    expect(
      describeToolCall(
        withMeta('kungal__get_engine', { id: 11, mobile: true }, {}),
        new Map([['kungal:engine-11', 'KiriKiri']])
      )
    ).toBe('已查看引擎「KiriKiri」的作品')
    expect(
      describeToolCall(
        found('kungal__search_toolsets', { q: '转区' }, 1),
        new Map()
      )
    ).toBe('已搜索 Gal 工具「转区」 -> 1 个')
  })

  it('names past conversations it searched and read', () => {
    const conversations = [
      { id: '41', title: '催泪作品推荐', updated_at: '2026-10-20T08:00:00Z' }
    ]
    expect(
      describeToolCall(
        withMeta(
          'search_conversations',
          { query: '催泪' },
          { conversations },
          1
        ),
        new Map()
      )
    ).toBe('已搜索过去的对话「催泪」 -> 1 个')
    expect(
      describeToolCall(
        withMeta(
          'search_conversations',
          { query: '催泪' },
          { conversations: [] },
          0
        ),
        new Map()
      )
    ).toBe('已搜索过去的对话「催泪」，没有找到')
    expect(
      describeToolCall(
        withMeta('read_conversation', { id: '41' }, { conversations }),
        new Map()
      )
    ).toBe('已阅读过去的对话「催泪作品推荐」')
    expect(
      describeToolCall(
        { ...tool('m', false), name: 'read_conversation', input: { id: '41' } },
        new Map()
      )
    ).toBe('正在阅读过去的对话')
  })

  it('says what was searched and how much came back', () => {
    expect(describeToolCall(tool('c'), new Map())).toBe(
      '已检索鲲 Galgame 论坛数据「猫耳 妹妹」 -> 3 部'
    )
    expect(describeToolCall(tool('c', false), new Map())).toBe(
      '正在检索鲲 Galgame 论坛数据「猫耳 妹妹」'
    )
    expect(
      describeToolCall(
        { ...tool('c'), name: 'get_work', input: '{"id":"3032"}' },
        new Map([['work-3032', '私の親愛なる妹']])
      )
    ).toBe('已查看作品「私の親愛なる妹」')
  })

  it('names the entity kind it looked up', () => {
    expect(
      describeToolCall(
        {
          ...tool('c'),
          name: 'search_entities',
          input: { kind: 'person', query: '田村ゆかり' }
        },
        new Map()
      )
    ).toBe('已查找人物「田村ゆかり」 -> 3 个')
    expect(
      describeToolCall(
        {
          ...tool('c'),
          name: 'get_entity',
          input: { kind: 'character', id: '3032' }
        },
        new Map([
          ['work-3032', '私の親愛なる妹'],
          ['character-3032', '朝武芳乃']
        ])
      )
    ).toBe('已查看角色「朝武芳乃」')
  })

  it('names the tags, traits and sort a browse used', () => {
    const done = (
      name: string,
      input: unknown,
      out: Partial<AgentToolOutput>
    ): AgentToolCall => ({
      ...tool('c'),
      name,
      input,
      output: {
        cards: 8,
        sources: 0,
        entities: [],
        links: [],
        total: null,
        meta: {},
        ...out
      }
    })
    expect(
      describeToolCall(
        done(
          'search_tags',
          { keywords: ['催泪', '虐心'], for: 'works' },
          { meta: { for: 'works', matched: ['催泪', '致郁'] } }
        ),
        new Map()
      )
    ).toBe('已查找标签「催泪」「虐心」 -> 催泪、致郁')
    expect(
      describeToolCall(
        done(
          'search_tags',
          { keywords: ['银发'], for: 'characters' },
          { cards: 0, meta: { for: 'characters', matched: [] } }
        ),
        new Map()
      )
    ).toBe('已查找角色特征「银发」，没有匹配的角色特征')
    expect(
      describeToolCall(
        done(
          'browse_works',
          { tag_ids: ['1'], sort: 'classic' },
          { total: 22, meta: { sort: 'classic', tags: ['催泪'] } }
        ),
        new Map()
      )
    ).toBe('已按标签「催泪」筛选经典作品 -> 8 部 / 共 22 部')
    expect(
      describeToolCall(
        done(
          'browse_characters',
          { trait_ids: ['1', '2'] },
          { total: 1589, meta: { traits: ['灰色', '傲娇'] } }
        ),
        new Map()
      )
    ).toBe('已按特征「灰色」「傲娇」筛选角色 -> 8 个 / 共 1589 个')
  })
})

describe('graftTurn', () => {
  it('adds an edit as a sibling of the message it replaces', () => {
    const thread = [msg('1'), msg('2'), msg('3'), msg('4')]
    graftTurn(thread, '2', [msg('5'), msg('6')])
    expect(thread.map((m) => m.id)).toEqual(['1', '2', '5', '6'])
    expect(thread[2]!.sibling_ids).toEqual(['3', '5'])
  })

  it('starts a new root branch from "0"', () => {
    const thread = [msg('1', ['1']), msg('2')]
    graftTurn(thread, '0', [msg('7'), msg('8')])
    expect(thread.map((m) => m.id)).toEqual(['7', '8'])
    expect(thread[0]!.sibling_ids).toEqual(['1', '7'])
  })

  it('appends a plain follow-up with no siblings', () => {
    const thread = [msg('1'), msg('2')]
    graftTurn(thread, '2', [msg('3'), msg('4')])
    expect(thread[2]!.sibling_ids).toEqual(['3'])
  })

  it('regenerates an answer that was itself added in this session', () => {
    const thread: AgentMessage[] = []
    graftTurn(thread, null, [msg('1'), msg('2')])
    graftTurn(thread, '1', [msg('3')])
    expect(thread.map((m) => m.id)).toEqual(['1', '3'])
    expect(thread[1]!.sibling_ids).toEqual(['2', '3'])
  })
})

describe('describeToolCall on a failed call', () => {
  const failed = (name: string, input: unknown, error: string) => ({
    ...tool('f'),
    name,
    input,
    output: null,
    error
  })

  it('says what failed and why', () => {
    expect(
      describeToolCall(
        failed('web_search', { query: '动画' }, '暂时无法使用'),
        new Map()
      )
    ).toBe('联网搜索「动画」失败：暂时无法使用')
    expect(
      describeToolCall(
        failed('kungal__search_topics', { query: '心跳' }, '查询超时'),
        new Map()
      )
    ).toBe('搜索论坛话题「心跳」失败：查询超时')
    expect(
      describeToolCall(failed('unknown', {}, '暂时无法使用'), new Map())
    ).toBe('查资料失败：暂时无法使用')
  })
})
