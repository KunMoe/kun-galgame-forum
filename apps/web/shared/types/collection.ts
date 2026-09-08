import type { GalgameCard } from './galgame'

export type CollectionVisibility = 'public' | 'private'

export interface CollectionUserBrief {
  id: number
  name: string
  avatar: string
}

export interface CollectionSummary {
  id: number
  name: string
  description: string
  visibility: CollectionVisibility
  is_default: boolean
  item_count: number
  preview_covers: string[]
  created: string
  updated: string
}

export interface CollectionDetail {
  id: number
  name: string
  description: string
  visibility: CollectionVisibility
  is_default: boolean
  item_count: number
  is_owner: boolean
  owner: CollectionUserBrief
  galgames: GalgameCard[]
  total: number
  created: string
  updated: string
}

export interface MyCollectionForGalgame {
  id: number
  name: string
  visibility: CollectionVisibility
  is_default: boolean
  item_count: number
  contains: boolean
}
