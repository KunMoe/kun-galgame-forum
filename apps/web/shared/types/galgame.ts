import type { GalgameEngineItem } from './galgame-engine'
import type { GalgameOfficialItem } from './galgame-official'
import type { GalgameDetailSeriesRef } from './galgame-series'
import type { GalgameTagItem } from './galgame-tag'
import type { GalgameRatingCardOnGalgamePage } from './galgame-rating'

export interface GalgameDetailTag extends GalgameTagItem {
  spoiler_level: number
}

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

export interface GalgameDetailStaff {
  role_key: string
  role_name: string
  people: GalgameDetailStaffName[]
}

export interface GalgameDetailStaffName {
  id: number
  name: string
  latin?: string
  characters?: string[]
}

export interface GalgameArtMeta {
  width: number
  height: number
  thumbhash?: string
}

export interface GalgameDetailCharacter {
  id: number
  name: string
  name_original?: string
  latin?: string
  kind: string
  spoiler: number
  identity?: string
  image?: string
  figure?: string
  image_meta?: GalgameArtMeta
  figure_meta?: GalgameArtMeta
  voices: GalgameDetailCharacterVoice[]
}

export interface GalgameDetailCharacterVoice {
  id: number
  name: string
  lang?: string
  latin?: string
}

export interface GalgameRatingBucket {
  score: number
  count: number
}

export interface GalgameRatingStats {
  average?: number
  stdev?: number
  min?: number
  max?: number
}

export interface GalgameExternalRating {
  source: string
  score: number
  vote_count: number
  rank?: number
  distribution?: GalgameRatingBucket[]
  stats?: GalgameRatingStats
}

export interface GalgamePlaytime {
  source: string
  minutes: number
  vote_count: number
}

export interface GalgameMyPlaytime {
  minutes: number
  status: string
}

export interface GalgameIntro {
  lang: string
  intro: string
  machine: boolean
}

export interface GalgameDetail {
  id: number
  moved_to?: number
  vndb_id: string
  user: KunUser
  name: string
  name_original: string
  introduction: GalgameIntro[]
  content_limit: string
  intro_text: string
  resource_update_time: Date | string
  release_date: string | null
  release_date_tba: boolean
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
  covers: GalgameCover[]
  screenshots: GalgameScreenshot[]
  view: number
  is_on_forum?: boolean
  indexed?: boolean
  status?: number
  original_language: string
  age_limit: 'all' | 'r18'
  platform: string[]
  language: string[]
  type: string[]
  contributor: KunUser[]
  like_count: number
  is_liked: boolean
  favorite_count: number
  is_favorited: boolean
  resource_publish_banned: boolean
  dlsite_purchase_url?: string
  dlsite_coupon_url?: string
  dlsite_campaign_name?: string
  alias: string[]
  engine: GalgameEngineItem[]
  official: GalgameOfficialItem[]
  series: GalgameDetailSeriesRef[]
  tag: GalgameDetailTag[]
  staff: GalgameDetailStaff[]
  characters: GalgameDetailCharacter[]
  ratings: GalgameRatingCardOnGalgamePage[]
  rating?: number
  rating_count?: number
  external_ratings?: GalgameExternalRating[]
  playtimes?: GalgamePlaytime[]
  my_playtime?: GalgameMyPlaytime | null
  refs?: Record<string, string>
  created: Date | string
  updated: Date | string
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
  catalog_id?: number
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
  product_work_id: number | null
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

export interface GalgameCalendarMeta {
  prev_month: string
  next_month: string
  has_prev: boolean
  has_next: boolean
  min_month: string
  max_month: string
  count: number
}

export interface GalgameCalendarMonth {
  month: string
  today: string
  items: GalgameCard[]
  meta: GalgameCalendarMeta
}

export interface GalgameCalendarPending {
  year: string
  items: GalgameCard[]
  count: number
}

export interface GalgameCalendarTBA {
  items: GalgameCard[]
  count: number
}

export interface GalgameCalendarUpcomingMonth {
  month: string
  items: GalgameCard[]
}

export interface GalgameCalendarUpcoming {
  today: string
  months: GalgameCalendarUpcomingMonth[]
  count: number
}
