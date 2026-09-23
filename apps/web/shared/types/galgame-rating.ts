export interface GalgameRatingCard {
  id: number
  user: KunUser
  recommend: string
  overall: number
  view: number
  galgame_type: string[]
  play_status: string
  short_summary: string

  art: number
  story: number
  music: number
  character: number
  route: number
  system: number
  voice: number
  replay_value: number
  spoiler_level: string

  like_count: number
  created: Date | string
  updated: Date | string

  galgame: {
    id: number
    name: string
    content_limit: string
  }
}

export interface GalgameRatingCardOnGalgamePage extends GalgameRatingCard {
  is_liked: boolean
}
