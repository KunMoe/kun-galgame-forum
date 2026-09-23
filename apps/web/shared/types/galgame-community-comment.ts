export interface CommunityWallState {
  anchor_kind: number
  anchor_id: string
  thread_id: number
  following: boolean
  notification_level: number
}

export interface CommunityFollowItem {
  anchor_kind: number
  anchor_id: string
  link: string
  title: string
  label: string
  galgame_id?: number
}

export interface CommunityFollowList {
  items: CommunityFollowItem[]
  next_cursor: string
}
