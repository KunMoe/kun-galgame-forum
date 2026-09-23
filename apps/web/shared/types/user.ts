import type { GalgameCard } from './galgame'

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
