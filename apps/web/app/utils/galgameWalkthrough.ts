import type {
  Image,
  WalkthroughChart,
  WalkthroughCharacter,
  WalkthroughNode,
  WalkthroughSegment,
  WalkthroughStep,
  WorkCharacter
} from '#shared/utils/api/schemas'
import type {
  GalgameWalkthroughGraphNode,
  GalgameWalkthroughNodeKind,
  GalgameWalkthroughRoute
} from '#shared/types/galgame-walkthrough'
import type { KunWalkthroughTextMode } from '~/constants/galgameWalkthrough'

export const COMMON_ROUTE_KEY = '__common__'

type Localized = Record<string, { value: string; is_machine: boolean }>

export const walkthroughText = (
  original: string,
  localized: Localized | undefined,
  mode: KunWalkthroughTextMode,
  textLang = 'ja'
) => {
  const text = original.normalize('NFKC')
  const zh = localized?.['zh-Hans']?.value.normalize('NFKC')
  if (mode === 'ja' || !zh || zh === text) {
    return { main: text, sub: '', lang: textLang }
  }
  return { main: zh, sub: mode === 'both' ? text : '', lang: 'zh-Hans' }
}

// Catalog translates save and entry labels since spec 2.76.0, but charts
// imported before it carry {} until infra re-applies them; until then the
// label is rewritten here.
export const walkthroughPointLabel = (
  label: string,
  localized: Localized | undefined,
  mode: KunWalkthroughTextMode
) => {
  const text = label.normalize('NFKC')
  if (mode === 'ja') return text
  const zh = localized?.['zh-Hans']?.value
  if (zh) return zh.normalize('NFKC')
  const save = text.match(/^セーブ\s*(\d+)\s*(.*)$/)
  if (!save) return text
  const note = save[2]!.replace(/^\((\d+)回使用\)$/, '(使用 $1 次)')
  return `存档 ${save[1]}${note ? ` ${note}` : ''}`
}

export const nodeLocalized = (
  chart: WalkthroughChart,
  node: WalkthroughNode
): Localized => {
  const find = <T extends { key: string; localized: Localized }>(list: T[]) =>
    list.find((x) => x.key === node.node_key)?.localized ?? {}
  switch (node.node_type) {
    case 'entry':
      return find(chart.entry_points)
    case 'save':
      return find(chart.save_points)
    case 'ending':
      return find(chart.chart_endings)
    default:
      return {}
  }
}

// A forum API deployed before this web build omits the field, and reading
// .length on it blanked the walkthrough page.
export const requiredEndingKeys = (segment: WalkthroughSegment): string[] =>
  segment.requires_ending_keys ?? []

export type WalkthroughStepKind = WalkthroughStep['step_type']

const STEP_KINDS = new Set<string>([
  'choice',
  'action',
  'save',
  'heading',
  'note'
])

// Catalog parses day headers and ※ notes into heading / note since spec
// 2.76.0, but charts imported before it still type them as choices (77 of
// the 794 "choices" in walkthrough 1039 are dates) until infra re-applies
// them. Catalog asks for an unknown step_type to be shown unnumbered.
export const walkthroughStepKind = (
  step: WalkthroughStep
): WalkthroughStepKind => {
  if (!STEP_KINDS.has(step.step_type)) return 'note'
  if (step.step_type !== 'choice' || !step.step_text) return step.step_type
  const text = step.step_text.normalize('NFKC')
  if (/^\d{1,2}月\d{1,2}日$/.test(text)) return 'heading'
  if (text.startsWith('※')) return 'note'
  return 'choice'
}

export type WalkthroughStepGroup = {
  heading: WalkthroughStep | null
  items: { step: WalkthroughStep; kind: WalkthroughStepKind; no: number }[]
}

export const groupWalkthroughSteps = (steps: WalkthroughStep[]) => {
  const groups: WalkthroughStepGroup[] = []
  let current: WalkthroughStepGroup | null = null
  let no = 0
  for (const step of steps) {
    const kind = walkthroughStepKind(step)
    if (kind === 'heading') {
      current = { heading: step, items: [] }
      groups.push(current)
      no = 0
      continue
    }
    if (kind !== 'save' && !step.step_text) continue
    if (!current) {
      current = { heading: null, items: [] }
      groups.push(current)
    }
    current.items.push({ step, kind, no: kind === 'choice' ? ++no : 0 })
  }
  return groups.filter((g) => g.items.length)
}

export const choiceCount = (steps: WalkthroughStep[]) =>
  steps.filter((s) => s.step_text && walkthroughStepKind(s) === 'choice').length

export const hasMachineTranslation = (chart: WalkthroughChart) => {
  const maps: (Localized | undefined)[] = [
    ...chart.route_sections.map((s) => s.localized),
    ...chart.entry_points.map((e) => e.localized),
    ...chart.save_points.map((s) => s.localized),
    ...chart.chart_endings.map((e) => e.localized),
    ...chart.chart_segments.flatMap((s) => [
      s.condition_localized,
      ...s.steps.flatMap((st) => [
        st.localized,
        ...st.alternatives.map((a) => a.localized)
      ])
    ])
  ]
  return maps.some((m) => m?.['zh-Hans']?.is_machine)
}

export const walkthroughPortraits = (
  characters: WalkthroughCharacter[],
  roster: WorkCharacter[]
) => {
  const byId = new Map(roster.map((c) => [c.id, c]))
  const out = new Map<string, Image>()
  for (const c of characters) {
    const member = c.catalog_character
      ? byId.get(c.catalog_character.id)
      : undefined
    const image = member?.image
    if (image) {
      out.set(c.key, image)
    }
  }
  return out
}

export const characterTones = (characters: WalkthroughCharacter[]) =>
  new Map(characters.map((c, i) => [c.key, i]))

export const segmentCharacterKey = (
  chart: WalkthroughChart,
  segment: WalkthroughSegment
) => {
  const ending =
    segment.to_node.node_type === 'ending'
      ? chart.chart_endings.find((e) => e.key === segment.to_node.node_key)
      : undefined
  if (ending?.character_keys.length === 1) {
    return ending.character_keys[0]!
  }
  const section = chart.route_sections.find(
    (s) => s.key === segment.section_key
  )
  if (section?.character_keys.length === 1) {
    return section.character_keys[0]!
  }
  return null
}

export const buildRoutes = (
  chart: WalkthroughChart,
  characters: WalkthroughCharacter[]
): GalgameWalkthroughRoute[] => {
  const tones = characterTones(characters)
  const byKey = new Map<string, WalkthroughSegment[]>()
  for (const segment of chart.chart_segments) {
    const key = segmentCharacterKey(chart, segment) ?? COMMON_ROUTE_KEY
    byKey.set(key, [...(byKey.get(key) ?? []), segment])
  }
  const routes: GalgameWalkthroughRoute[] = characters
    .filter((c) => byKey.has(c.key))
    .map((c) => ({
      key: c.key,
      name: c.source_name,
      tone: tones.get(c.key) ?? null,
      segments: byKey.get(c.key)!
    }))
  if (byKey.has(COMMON_ROUTE_KEY)) {
    routes.push({
      key: COMMON_ROUTE_KEY,
      name: '共通 / 其他',
      tone: null,
      segments: byKey.get(COMMON_ROUTE_KEY)!
    })
  }
  return routes
}

export const saveCreators = (chart: WalkthroughChart) => {
  const out = new Map<string, string>()
  for (const segment of chart.chart_segments) {
    for (const step of segment.steps) {
      if (
        step.step_type === 'save' &&
        step.save_key &&
        !out.has(step.save_key)
      ) {
        out.set(step.save_key, segment.key)
      }
    }
  }
  return out
}

export const nodeId = (node: WalkthroughNode, segmentKey: string) =>
  node.node_type === 'collect' || node.node_type === 'open'
    ? `${node.node_type}:${segmentKey}`
    : `${node.node_type}:${node.node_key}`

export const NODE_SIZE: Record<
  GalgameWalkthroughNodeKind,
  { width: number; height: number }
> = {
  entry: { width: 112, height: 40 },
  save: { width: 104, height: 36 },
  ending: { width: 192, height: 48 },
  collect: { width: 88, height: 30 },
  open: { width: 56, height: 28 }
}

export const nodeLabel = (
  chart: WalkthroughChart,
  node: WalkthroughNode
): string => {
  switch (node.node_type) {
    case 'entry':
      return (
        chart.entry_points.find((e) => e.key === node.node_key)?.label ??
        'START'
      )
    case 'save':
      return (
        chart.save_points.find((s) => s.key === node.node_key)?.label ?? '存档'
      )
    case 'ending':
      return (
        chart.chart_endings.find((e) => e.key === node.node_key)?.ending_name ??
        'END'
      )
    case 'collect':
      return '回收后结束'
    default:
      return '……'
  }
}

export type WalkthroughGraphModel = {
  nodes: Omit<GalgameWalkthroughGraphNode, 'x' | 'y'>[]
  edges: {
    id: string
    from: string
    to: string
    segmentKey: string
    steps: WalkthroughSegment['steps']
    tone: number | null
  }[]
}

export const buildGraphModel = (
  chart: WalkthroughChart,
  characters: WalkthroughCharacter[]
): WalkthroughGraphModel => {
  const tones = characterTones(characters)
  const nodes = new Map<string, Omit<GalgameWalkthroughGraphNode, 'x' | 'y'>>()
  const edges: WalkthroughGraphModel['edges'] = []
  const addNode = (
    id: string,
    kind: GalgameWalkthroughNodeKind,
    key: string | null,
    label: string,
    tone: number | null
  ) => {
    const existing = nodes.get(id)
    if (existing) {
      existing.tone ??= tone
      return
    }
    const localized = nodeLocalized(chart, { node_type: kind, node_key: key })
    nodes.set(id, {
      id,
      kind,
      key,
      label,
      localized,
      tone,
      contextKey: null,
      ...NODE_SIZE[kind]
    })
  }

  for (const segment of chart.chart_segments) {
    const routeKey = segmentCharacterKey(chart, segment)
    const tone = routeKey ? (tones.get(routeKey) ?? null) : null
    let current = nodeId(segment.from_node, segment.key)
    addNode(
      current,
      segment.from_node.node_type,
      segment.from_node.node_key ?? null,
      nodeLabel(chart, segment.from_node),
      null
    )
    let pending: WalkthroughSegment['steps'] = []
    let piece = 0
    const close = (to: string) => {
      edges.push({
        id: `${segment.key}:${piece++}`,
        from: current,
        to,
        segmentKey: segment.key,
        steps: pending,
        tone
      })
      pending = []
      current = to
    }
    for (const step of segment.steps) {
      if (step.step_type === 'save' && step.save_key) {
        const saveNode = { node_type: 'save', node_key: step.save_key } as const
        const id = nodeId(saveNode, segment.key)
        addNode(id, 'save', step.save_key, nodeLabel(chart, saveNode), tone)
        close(id)
        continue
      }
      pending = [...pending, step]
    }
    const to = nodeId(segment.to_node, segment.key)
    const ending =
      segment.to_node.node_type === 'ending'
        ? chart.chart_endings.find((e) => e.key === segment.to_node.node_key)
        : undefined
    const endingTone = ending?.character_keys.length
      ? (tones.get(ending.character_keys[0]!) ?? tone)
      : tone
    addNode(
      to,
      segment.to_node.node_type,
      segment.to_node.node_key ?? null,
      nodeLabel(chart, segment.to_node),
      endingTone
    )
    if (ending && !ending.character_keys.length) {
      nodes.get(to)!.contextKey ??= routeKey
    }
    close(to)
  }
  for (const entry of chart.entry_points) {
    if (!entry.unlocked_by_ending_key) continue
    const from = `ending:${entry.unlocked_by_ending_key}`
    const to = `entry:${entry.key}`
    if (nodes.has(from) && nodes.has(to)) {
      edges.push({
        id: `unlock:${entry.key}`,
        from,
        to,
        segmentKey: '',
        steps: [],
        tone: null
      })
    }
  }
  return { nodes: [...nodes.values()], edges }
}
