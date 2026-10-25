import { describe, it, expect } from 'vitest'
import type {
  WalkthroughChart,
  WalkthroughCharacter
} from '#shared/utils/api/schemas'
import {
  buildGraphModel,
  buildRoutes,
  COMMON_ROUTE_KEY,
  saveCreators,
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
    { key: 'start', label: 'START', unlocked_by_ending_key: null }
  ],
  save_points: [
    { key: 'sv01', label: 'セーブ01' },
    { key: 'sv02', label: 'セーブ02' }
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
      steps: [choice('d')],
      to_node: { node_type: 'ending', node_key: 'e2' }
    },
    {
      key: 'g3',
      section_key: null,
      from_node: { node_type: 'save', node_key: 'sv02' },
      condition: null,
      condition_localized: {},
      steps: [choice('e')],
      to_node: { node_type: 'ending', node_key: 'e3' }
    },
    {
      key: 'g4',
      section_key: 's2',
      from_node: { node_type: 'save', node_key: 'sv02' },
      condition: null,
      condition_localized: {},
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
  it('shows the original first and the translation under it by default', () => {
    expect(walkthroughText('はい', zh, 'both')).toEqual({
      main: 'はい',
      sub: '是'
    })
    expect(walkthroughText('はい', zh, 'zh')).toEqual({ main: '是', sub: '' })
    expect(walkthroughText('はい', {}, 'zh')).toEqual({ main: 'はい', sub: '' })
  })
})
