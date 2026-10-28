import type {
  WalkthroughSegment,
  WalkthroughStep
} from '#shared/utils/api/schemas'

export type GalgameWalkthroughNodeKind =
  | 'entry'
  | 'save'
  | 'ending'
  | 'collect'
  | 'open'

export interface GalgameWalkthroughGraphNode {
  id: string
  kind: GalgameWalkthroughNodeKind
  key: string | null
  label: string
  localized: Record<string, { value: string; is_machine: boolean }>
  tone: number | null
  contextKey: string | null
  x: number
  y: number
  width: number
  height: number
}

export interface GalgameWalkthroughGraphEdge {
  id: string
  from: string
  to: string
  segmentKey: string
  steps: WalkthroughStep[]
  tone: number | null
  path: string
  head: { x: number; y: number; angle: number }
  labelX: number
  labelY: number
}

export interface GalgameWalkthroughGraphLayout {
  nodes: GalgameWalkthroughGraphNode[]
  edges: GalgameWalkthroughGraphEdge[]
  width: number
  height: number
}

export interface GalgameWalkthroughRoute {
  key: string
  name: string
  tone: number | null
  segments: WalkthroughSegment[]
}
