import type { MessageStatus } from '~~/shared/types/utils/message'
import type { KunFeedTab } from '~/constants/activity'

export interface KUNGalgameSettingsStore {
  feedTabs: KunFeedTab[]
  feedTabsVersion: number
  showKUNGalgamePageTransparency: number
  showKUNGalgameFontStyle: string
  showKUNGalgameContentLimit: string
  showKUNGalgamePreferOriginalName: boolean
  showKUNGalgameBackground: number
  showKUNGalgameBackgroundBlur: number
  showKUNGalgameBackgroundBrightness: number
  showKUNGalgameBackgroundOpacity: number
  showKUNGalgameBackLoli: boolean
  showKUNGalgameNoResource: boolean
  showKUNGalgameRounded: 'none' | 'sm' | 'md' | 'lg'
  showKUNGalgamePhoneColumns: 2 | 3
  showKUNGalgameGallerySexualLevels: number[]
  showKUNGalgameGalleryViolenceLevels: number[]
}

export interface TempSettingStore {
  showKUNGalgameHamburger: boolean
  showKUNGalgamePanel: boolean
  showKUNGalgameUserPanel: boolean

  showKUNGalgameMessageBox: boolean
  showKUNGalgameMoemoepointLog: boolean
  showKUNGalgameLogout: boolean
  showKUNGalgameCreatorApply: boolean
  messageStatus: MessageStatus
}
