import { defineStore } from 'pinia'
import { ref } from 'vue'
import { withImageVariant } from '../../../shared/utils/getEffectiveBanner'
import type { UserStore } from '../types/user'

export const usePersistUserStore = defineStore(
  'KUNGalgameUser',
  () => {
    const id = ref<UserStore['id']>(0)
    const sub = ref<UserStore['sub']>('')
    const name = ref<UserStore['name']>('')
    const avatar = ref<UserStore['avatar']>('')
    const avatarMin = ref<UserStore['avatarMin']>('')
    const moemoepoint = ref<UserStore['moemoepoint']>(0)
    const roles = ref<UserStore['roles']>([])
    const isCheckIn = ref<UserStore['isCheckIn']>(false)
    const dailyToolsetUploadBytes = ref<UserStore['dailyToolsetUploadBytes']>(0)
    const adultConfirmed = ref<UserStore['adultConfirmed']>(false)
    const nsfwDisplay = ref<UserStore['nsfwDisplay']>('hide')

    const setUserInfo = (user: UserStore) => {
      id.value = user.id
      sub.value = user.sub
      name.value = user.name
      avatar.value = user.avatar
      avatarMin.value = withImageVariant(user.avatar, '100')
      moemoepoint.value = user.moemoepoint
      roles.value = user.roles
      isCheckIn.value = user.isCheckIn
      dailyToolsetUploadBytes.value = user.dailyToolsetUploadBytes
      adultConfirmed.value = user.adultConfirmed
      nsfwDisplay.value = user.nsfwDisplay
    }

    const setProfileInfo = (profile: {
      name: string
      avatar: string
      roles: string[]
      adultConfirmed?: boolean
      nsfwDisplay?: string
    }) => {
      name.value = profile.name
      avatar.value = profile.avatar
      avatarMin.value = profile.avatar
        ? withImageVariant(profile.avatar, '100')
        : ''
      roles.value = profile.roles
      adultConfirmed.value = profile.adultConfirmed ?? false
      nsfwDisplay.value = profile.nsfwDisplay ?? 'hide'
    }

    const setContentStance = (confirmed: boolean, display: string) => {
      adultConfirmed.value = confirmed
      nsfwDisplay.value = display
    }

    const resetUser = () => {
      id.value = 0
      sub.value = ''
      name.value = ''
      avatar.value = ''
      avatarMin.value = ''
      moemoepoint.value = 0
      roles.value = []
      isCheckIn.value = false
      dailyToolsetUploadBytes.value = 0
      adultConfirmed.value = false
      nsfwDisplay.value = 'hide'
    }

    return {
      id,
      sub,
      name,
      avatar,
      avatarMin,
      moemoepoint,
      roles,
      isCheckIn,
      dailyToolsetUploadBytes,
      adultConfirmed,
      nsfwDisplay,
      setUserInfo,
      setProfileInfo,
      setContentStance,
      resetUser
    }
  },
  {
    persist: true
  }
)
