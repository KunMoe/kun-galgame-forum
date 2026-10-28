import dagre from '@dagrejs/dagre'
import type { GalgameWalkthroughGraphLayout } from '#shared/types/galgame-walkthrough'
import type { WalkthroughGraphModel } from './galgameWalkthrough'

const MARGIN = 24
export const WALKTHROUGH_HEAD_LENGTH = 9
export const WALKTHROUGH_EDGE_LABEL = { width: 88, height: 26 }

type Point = { x: number; y: number }

const smoothPath = (points: Point[]) => {
  const [first, ...rest] = points
  if (!first) return ''
  if (rest.length === 0) return `M ${first.x} ${first.y}`
  let d = `M ${first.x} ${first.y}`
  for (let i = 0; i < rest.length; i++) {
    const p = rest[i]!
    const prev = i === 0 ? first : rest[i - 1]!
    const midY = (prev.y + p.y) / 2
    d += ` C ${prev.x} ${midY}, ${p.x} ${midY}, ${p.x} ${p.y}`
  }
  return d
}

export const layoutWalkthroughGraph = (
  model: WalkthroughGraphModel
): GalgameWalkthroughGraphLayout => {
  const g = new dagre.graphlib.Graph({ multigraph: true })
  g.setGraph({
    rankdir: 'TB',
    nodesep: 24,
    ranksep: 28,
    edgesep: 8,
    marginx: MARGIN,
    marginy: MARGIN
  })
  g.setDefaultEdgeLabel(() => ({}))
  for (const n of model.nodes) {
    g.setNode(n.id, { width: n.width, height: n.height })
  }
  for (const e of model.edges) {
    g.setEdge(
      e.from,
      e.to,
      e.steps.length
        ? { minlen: 1, weight: 2, labelpos: 'c', ...WALKTHROUGH_EDGE_LABEL }
        : { minlen: 1, weight: 1 },
      e.id
    )
  }
  dagre.layout(g)

  const nodes = model.nodes.map((n) => {
    const p = g.node(n.id)
    return { ...n, x: p.x, y: p.y }
  })
  const edges = model.edges.map((e) => {
    const label = g.edge({ v: e.from, w: e.to, name: e.id })
    const points = (label?.points ?? []) as Point[]
    const last = points.at(-1) ?? { x: 0, y: 0 }
    const before = points.at(-2) ?? last
    const angle =
      (Math.atan2(last.y - before.y, last.x - before.x) * 180) / Math.PI
    const mid = points[Math.floor(points.length / 2)] ?? last
    return {
      ...e,
      path: smoothPath(points),
      head: { x: last.x, y: last.y, angle },
      labelX: label?.x ?? mid.x,
      labelY: label?.y ?? mid.y
    }
  })
  const graph = g.graph()
  return {
    nodes,
    edges,
    width: Math.ceil(graph.width ?? 0),
    height: Math.ceil(graph.height ?? 0)
  }
}
