import type { GalgameCard } from './galgame'

export interface UserTopic {
  id: number
  title: string
  created: Date | string
}

export type UserGalgame = GalgameCard

export interface UserGalgameResource {
  id: number
  galgame_id: number
  galgame_name: string
  type: string
  language: string
  platform: string
  size: string
  link: string[]
  code: string
  password: string
  note: string
  status: number
  created: Date | string
}

export interface UserReply {
  topic_id: number
  floor: number
  content: string
  created: Date | string
}

export interface UserComment {
  id: number
  topic_id: number
  content: string
  created: Date | string
}
