import type { GalgameCard } from './galgame'

export interface GalgameSeriesSample {
  name: string
  effective_banner_hash?: string
  effective_banner_url?: string
  effective_banner_thumbhash?: string
}

export interface GalgameSeriesCard {
  id: number
  name: string
  is_nsfw: boolean
  galgame_count: number
  catalog_galgame_count: number
  sample_galgame: GalgameSeriesSample[]
}

export interface GalgameSeriesDetail {
  id: number
  name: string
  description: string
  galgame: GalgameCard[]
  galgame_count: number
}

export interface GalgameDetailSeriesRef {
  id: number
  name: string
}
