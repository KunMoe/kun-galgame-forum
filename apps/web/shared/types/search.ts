import type {
  CommentSearchHit,
  GalgameResource,
  ReplySearchHit,
  TopicSummary,
  UserSearchHit,
  WorkRef
} from '../utils/api/schemas'
import type { ToolsetCard } from './toolset'

export type SearchResultToolset = ToolsetCard
export type SearchResultResource = GalgameResource

export type SearchEntityFamily =
  | 'character'
  | 'company'
  | 'staff'
  | 'tag'
  | 'series'
  | 'engine'

export interface SearchEntityItem {
  id: number
  family: SearchEntityFamily
  name: string
  alias?: string
  image?: string
  work_count?: number
}

export interface SearchEntityGroup {
  family: SearchEntityFamily
  total: number
  items: SearchEntityItem[]
}

export interface SearchEntityResult {
  groups: SearchEntityGroup[]
  total: number
}

export type SearchType =
  | 'all'
  | 'galgame'
  | 'topic'
  | 'entity'
  | 'resource'
  | 'user'
  | 'reply'
  | 'comment'
  | 'galcomment'
  | 'toolset'

export type SearchPagedType = Exclude<
  SearchType,
  'all' | 'entity' | 'galcomment'
>

export type SearchResult =
  | TopicSummary
  | WorkRef
  | SearchResultResource
  | SearchResultToolset
  | UserSearchHit
  | ReplySearchHit
  | CommentSearchHit

// Absent, not zero, for a lane that failed and for the comment walls, which
// answer no total: SearchNavCount draws nothing for absent.
export type SearchOverviewTotals = Partial<Record<SearchType, number>>
