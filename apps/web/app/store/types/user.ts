export interface UserStore {
  id: number
  sub: string
  name: string
  avatar: string
  avatarMin: string
  moemoepoint: number
  roles: string[]
  isCheckIn: boolean
  dailyToolsetUploadBytes: number
  adultConfirmed: boolean
  nsfwDisplay: string
}
