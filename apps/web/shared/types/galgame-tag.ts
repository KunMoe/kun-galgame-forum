import type { KunGalgameTagCategory } from '~/constants/galgameTag'

export interface GalgameTag {
  id: number
  name: string
  category: KunGalgameTagCategory
}

export interface GalgameTagItem {
  id: number
  name: string
  category: KunGalgameTagCategory
  galgame_count: number
}

export interface GalgameTaxonomySearchItem {
  id: number
  name: string
  logo?: string
}
