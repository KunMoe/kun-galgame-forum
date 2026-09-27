import type { UserRef } from '#shared/utils/api/schemas'
import { deletedUserName } from '#shared/utils/deletedUser'

export { deletedUserName }

export const toKunUser = (ref: UserRef): KunUser => ({
  id: Number(ref.id),
  name: ref.name ?? deletedUserName,
  avatar: ref.avatar?.url ?? '',
  avatarDecoration: ref.avatar_frame && {
    src: ref.avatar_frame.static_url,
    animatedSrc: ref.avatar_frame.animated_url ?? undefined
  }
})

export const toKunUserWithPoints = (
  ref: UserRef,
  moemoepoint: number
): KunUser & { moemoepoint: number } => ({
  ...toKunUser(ref),
  moemoepoint
})
