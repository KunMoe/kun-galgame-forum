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

export interface PlaytimeMineItem {
  galgame: GalgameCard
  minutes: number
  status: string
  clients: number
}

export interface PlaytimeMinePage {
  items: PlaytimeMineItem[]
  total: number
  total_minutes: number
  finished_works: number
  truncated: boolean
}

export interface UserClaimItem {
  work_id: number
  display_name: string
  site: string
  claim_state: string

  last_event_id: number
  last_from_state: string | null
  last_to_state: string
  last_reason: string | null
  last_actor_uid: number
  last_event_at: string

  first_acted_at: string
  acted_count: number
}

export interface UserClaimList {
  items: UserClaimItem[]
  next_before: number
  total: number
}


