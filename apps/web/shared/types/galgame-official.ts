import type { KunGalgameOfficialCategory } from '~/constants/galgameOfficial'

export interface GalgameOfficialItem {
  id: number
  name: string
  link: string
  category: KunGalgameOfficialCategory
  roles?: string[]
  lang: string
  alias: string[]
  galgame_count: number
  logo?: string
}

export type GalgameOfficialRelation =
  | 'parent'
  | 'subsidiary'
  | 'imprint'
  | 'imprint_of'
  | 'spawned'
  | 'origin'
  | 'succeeded_by'
  | 'formerly'

export interface GalgameOfficialRelationNode {
  id: number
  name: string
  logo: string
  work_count: number
}

export interface GalgameOfficialRelationEdge {
  from: number
  to: number
  relation: GalgameOfficialRelation
}

export interface GalgameOfficialRelationGraph {
  nodes: GalgameOfficialRelationNode[]
  edges: GalgameOfficialRelationEdge[]
}
