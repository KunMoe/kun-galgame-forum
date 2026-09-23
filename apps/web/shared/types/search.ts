import type { HomeTopic, HomeGalgame } from './home'
import type { ToolsetSummary } from '#shared/utils/api/schemas'
import type { GalgameResourceCard } from './galgame-resource'

export type SearchResultTopic = HomeTopic
export type SearchResultGalgame = HomeGalgame
export type SearchResultToolset = ToolsetSummary
export type SearchResultResource = GalgameResourceCard

export interface SearchResultUser extends KunUser {
  bio: string
  roles: string[]
  created: string | null
  topic_count: number
  reply_count: number
}

export interface SearchResultReply {
  id: number
  topic_id: number
  topic_title: string
  floor: number
  content: string
  user: KunUser
  created: Date | string
}

export type SearchResultComment = {
  id: number
  topic_id: number
  topic_title: string
  content: string
  user: KunUser
  target_user?: KunUser
  created: Date | string
}

// A comment wall on a Galgame or one of the resource pages around it. These
// live in the community service, which answers a cursor and no total — the lane
// therefore loads more rather than paginating, and carries no rail count.
export interface SearchResultGalComment {
  id: number
  content: string
  link: string
  title: string
  label: string
  galgame_id?: number
  user: KunUser
  created: string
}

export interface SearchGalCommentResult {
  items: SearchResultGalComment[]
  next_cursor: string
}

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
  | SearchResultTopic
  | SearchResultGalgame
  | SearchResultResource
  | SearchResultToolset
  | SearchResultUser
  | SearchResultReply
  | SearchResultComment

export interface SearchQuickTotals {
  topic: number
  galgame: number
  user: number
}

export interface SearchQuickResult {
  topics: SearchResultTopic[]
  galgames: SearchResultGalgame[]
  users: SearchResultUser[]
  totals: SearchQuickTotals
}

export interface SearchOverviewTotals {
  galgame: number
  topic: number
  entity: number
  resource: number
  user: number
  reply: number
  comment: number
  toolset: number
}

export interface SearchOverviewResult {
  topics: SearchResultTopic[]
  galgames: SearchResultGalgame[]
  entities: SearchEntityGroup[]
  resources: SearchResultResource[]
  users: SearchResultUser[]
  replies: SearchResultReply[]
  comments: SearchResultComment[]
  gal_comments: SearchResultGalComment[]
  toolsets: SearchResultToolset[]
  totals: SearchOverviewTotals
}
