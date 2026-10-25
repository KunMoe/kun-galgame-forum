import type {
  Image,
  WalkthroughChart,
  WalkthroughCharacter,
  WalkthroughNode,
  WalkthroughSegment,
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
  mode: KunWalkthroughTextMode
) => {
  const zh = localized?.['zh-Hans']?.value
  if (mode === 'ja' || !zh || zh === original) {
    return { main: original, sub: '', lang: 'ja' }
  }
  return { main: zh, sub: mode === 'both' ? original : '', lang: 'zh-Hans' }
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
  ending: { width: 176, height: 48 },
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
    const localized =
      kind === 'ending'
        ? (chart.chart_endings.find((e) => e.key === key)?.localized ?? {})
        : {}
    nodes.set(id, { id, kind, key, label, localized, tone, ...NODE_SIZE[kind] })
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
