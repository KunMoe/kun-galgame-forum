export interface GalgameCover {
  image_hash: string
  sort_order: number
  sexual: number
  violence: number
  source: string
  source_key: string
  kind?: string
  cdn_url?: string
  width?: number
  height?: number
  thumbhash?: string
  id?: number
  vote_count?: number
  voted?: boolean
}

export interface GalgameScreenshot extends GalgameCover {
  caption: string
}

export interface GalgameArtMeta {
  width: number
  height: number
  thumbhash?: string
}

export interface GalgameDetailCharacterVoice {
  id: number
  name: string
  lang?: string
  latin?: string
}

export interface GalgameCard {
  id: number
  name: string
  name_original: string
  user: KunUser
  content_limit: string
  view: number
  like_count: number
  rating?: number
  rating_count?: number
  is_on_forum?: boolean
  platform: string[]
  language: string[]
  resource_update_time: Date | string
  release_date?: string | null
  release_date_tba?: boolean
  release_precision?: 'day' | 'month' | 'year' | 'tba' | 'unknown'
  status?: number
  effective_banner_hash?: string
  effective_banner_url?: string
  effective_banner_width?: number
  effective_banner_height?: number
  effective_banner_thumbhash?: string
  effective_portrait_hash?: string
  effective_portrait_url?: string
  effective_portrait_width?: number
  effective_portrait_height?: number
  effective_portrait_thumbhash?: string
  company?: string
  via_official?: { id: number; name: string }
}
