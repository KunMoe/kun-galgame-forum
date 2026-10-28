import { describe, it, expect } from 'vitest'
import type {
  WalkthroughChart,
  WalkthroughCharacter
} from '#shared/utils/api/schemas'
import {
  buildGraphModel,
  buildRoutes,
  choiceCount,
  COMMON_ROUTE_KEY,
  groupWalkthroughSteps,
  hasMachineTranslation,
  saveCreators,
  walkthroughPointLabel,
  walkthroughText
} from './galgameWalkthrough'
import { layoutWalkthroughGraph } from './galgameWalkthroughLayout'

const choice = (text: string) => ({
  step_type: 'choice' as const,
  step_text: text,
  localized: {},
  alternatives: [],
  is_achievement_only: false,
  save_key: null
})
const save = (key: string) => ({
  step_type: 'save' as const,
  step_text: null,
  localized: {},
  alternatives: [],
  is_achievement_only: false,
  save_key: key
})

const characters: WalkthroughCharacter[] = [
  { key: 'c1', source_name: '寧々', catalog_character: null },
  { key: 'c2', source_name: 'めぐる', catalog_character: null }
]

const chart: WalkthroughChart = {
  route_sections: [
    { key: 's1', title: '綾地 寧々', localized: {}, character_keys: ['c1'] },
    { key: 's2', title: '因幡 めぐる', localized: {}, character_keys: ['c2'] }
  ],
  entry_points: [
    {
      key: 'start',
      label: 'START',
      localized: {},
      unlocked_by_ending_key: null
    }
  ],
  save_points: [
    { key: 'sv01', label: 'セーブ01', localized: {} },
    { key: 'sv02', label: 'セーブ02', localized: {} }
  ],
  chart_endings: [
    {
      key: 'e1',
      ending_name: '寧々END',
      localized: {},
      ending_type: 'heroine',
      character_keys: ['c1']
    },
    {
      key: 'e2',
      ending_name: 'めぐるEND',
      localized: {},
      ending_type: 'heroine',
      character_keys: ['c2']
    },
    {
      key: 'e3',
      ending_name: 'ノーマルEND',
      localized: {},
      ending_type: 'normal',
      character_keys: []
    }
  ],
  chart_segments: [
    {
      key: 'g1',
      section_key: 's1',
      from_node: { node_type: 'entry', node_key: 'start' },
      condition: null,
      condition_localized: {},
      requires_ending_keys: [],
      steps: [
        choice('a'),
        save('sv01'),
        choice('b'),
        save('sv02'),
        choice('c')
      ],
      to_node: { node_type: 'ending', node_key: 'e1' }
    },
    {
      key: 'g2',
      section_key: 's2',
      from_node: { node_type: 'save', node_key: 'sv01' },
      condition: null,
      condition_localized: {},
      requires_ending_keys: [],
      steps: [choice('d')],
      to_node: { node_type: 'ending', node_key: 'e2' }
    },
    {
      key: 'g3',
      section_key: null,
      from_node: { node_type: 'save', node_key: 'sv02' },
      condition: null,
      condition_localized: {},
      requires_ending_keys: [],
      steps: [choice('e')],
      to_node: { node_type: 'ending', node_key: 'e3' }
    },
    {
      key: 'g4',
      section_key: 's2',
      from_node: { node_type: 'save', node_key: 'sv02' },
      condition: null,
      condition_localized: {},
      requires_ending_keys: [],
      steps: [choice('f')],
      to_node: { node_type: 'collect', node_key: null }
    }
  ]
}

describe('buildRoutes', () => {
  it('groups segments by the character their ending or heading belongs to', () => {
    const routes = buildRoutes(chart, characters)
    expect(routes.map((r) => [r.key, r.segments.map((s) => s.key)])).toEqual([
      ['c1', ['g1']],
      ['c2', ['g2', 'g4']],
      [COMMON_ROUTE_KEY, ['g3']]
    ])
  })
})

describe('saveCreators', () => {
  it('names the segment that creates each save, so a resume can jump to it', () => {
    expect(saveCreators(chart)).toEqual(
      new Map([
        ['sv01', 'g1'],
        ['sv02', 'g1']
      ])
    )
  })
})

describe('buildGraphModel', () => {
  it('splits a segment at its save steps and shares save nodes across routes', () => {
    const model = buildGraphModel(chart, characters)
    const g1 = model.edges.filter((e) => e.segmentKey === 'g1')
    expect(g1.map((e) => [e.from, e.to, e.steps.length])).toEqual([
      ['entry:start', 'save:sv01', 1],
      ['save:sv01', 'save:sv02', 1],
      ['save:sv02', 'ending:e1', 1]
    ])
    expect(model.nodes.filter((n) => n.kind === 'save')).toHaveLength(2)
    expect(model.nodes.find((n) => n.id === 'collect:g4')).toBeTruthy()
    expect(model.nodes.find((n) => n.id === 'ending:e2')?.tone).toBe(1)
  })

  it('names the route of an ending that belongs to no character', () => {
    const generic = {
      ...chart,
      chart_endings: chart.chart_endings.map((e) =>
        e.key === 'e2' ? { ...e, character_keys: [] } : e
      )
    }
    const nodes = buildGraphModel(generic, characters).nodes
    expect(nodes.find((n) => n.id === 'ending:e2')?.contextKey).toBe('c2')
    expect(nodes.find((n) => n.id === 'ending:e1')?.contextKey).toBeNull()
  })

  it('lays out every node inside the canvas', () => {
    const layout = layoutWalkthroughGraph(buildGraphModel(chart, characters))
    for (const n of layout.nodes) {
      expect(n.x - n.width / 2).toBeGreaterThanOrEqual(0)
      expect(n.x + n.width / 2).toBeLessThanOrEqual(layout.width)
    }
    expect(layout.edges.every((e) => e.path.startsWith('M '))).toBe(true)
  })
})

describe('walkthroughText', () => {
  const zh = { 'zh-Hans': { value: '是', is_machine: true } }
  it('shows the translation first and the original in parentheses', () => {
    expect(walkthroughText('はい', zh, 'both')).toEqual({
      main: '是',
      sub: 'はい',
      lang: 'zh-Hans'
    })
    expect(walkthroughText('はい', zh, 'zh')).toEqual({
      main: '是',
      sub: '',
      lang: 'zh-Hans'
    })
    expect(walkthroughText('はい', zh, 'ja')).toEqual({
      main: 'はい',
      sub: '',
      lang: 'ja'
    })
    expect(walkthroughText('はい', {}, 'zh')).toEqual({
      main: 'はい',
      sub: '',
      lang: 'ja'
    })
  })

  it('shows a Chinese chart as written in every mode', () => {
    for (const mode of ['both', 'ja', 'zh'] as const) {
      expect(walkthroughText('选是', {}, mode, 'zh-Hans')).toEqual({
        main: '选是',
        sub: '',
        lang: 'zh-Hans'
      })
    }
  })
})

describe('hasMachineTranslation', () => {
  it('finds a machine translation anywhere in the chart', () => {
    expect(hasMachineTranslation(chart)).toBe(false)
    const human = { 'zh-Hans': { value: '宁宁END', is_machine: false } }
    const machine = { 'zh-Hans': { value: '宁宁END', is_machine: true } }
    const withEnding = (localized: typeof human) => ({
      ...chart,
      chart_endings: [{ ...chart.chart_endings[0]!, localized }]
    })
    expect(hasMachineTranslation(withEnding(human))).toBe(false)
    expect(hasMachineTranslation(withEnding(machine))).toBe(true)
  })
})

describe('walkthroughPointLabel', () => {
  const zh = { 'zh-Hans': { value: '存档１３', is_machine: true } }
  it('prefers the catalog translation and rewrites an untranslated save', () => {
    expect(walkthroughPointLabel('セーブ１３', zh, 'both')).toBe('存档13')
    expect(walkthroughPointLabel('セーブ１３', {}, 'both')).toBe('存档 13')
    expect(walkthroughPointLabel('セーブ１２（２回使用）', {}, 'zh')).toBe(
      '存档 12 (使用 2 次)'
    )
    expect(walkthroughPointLabel('セーブ１３', zh, 'ja')).toBe('セーブ13')
    expect(walkthroughPointLabel('鈴１', {}, 'both')).toBe('鈴1')
  })
})

describe('groupWalkthroughSteps', () => {
  it('groups under headings and numbers choices within each group', () => {
    const steps = [
      choice('はい'),
      { ...choice('第九話'), step_type: 'heading' as const },
      choice('見守る'),
      { ...choice('ＥＮＤを見ていれば「はい」'), step_type: 'note' as const },
      save('sv01'),
      choice('貸す'),
      choice('５月１５日'),
      choice('５月１６日'),
      choice('※ＥＮＤを見ている場合は「はい」'),
      choice('探す'),
      { ...choice('迷路'), step_type: 'maze' as unknown as 'choice' }
    ]
    const groups = groupWalkthroughSteps(steps)
    expect(
      groups.map((g) => [
        g.heading?.step_text ?? null,
        g.items.map((i) => `${i.kind}:${i.no}`)
      ])
    ).toEqual([
      [null, ['choice:1']],
      ['第九話', ['choice:1', 'note:0', 'save:0', 'choice:2']],
      ['５月１６日', ['note:0', 'choice:1', 'note:0']]
    ])
    expect(choiceCount(steps)).toBe(4)
  })
})
