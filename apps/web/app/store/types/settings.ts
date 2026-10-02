import type { MessageStatus } from '~~/shared/types/utils/message'
import type { KunFeedTab } from '~/constants/activity'

export interface KUNGalgameSettingsStore {
  feedTabs: KunFeedTab[]
  feedTabsVersion: number
  showKUNGalgamePageTransparency: number
  showKUNGalgameContentLimit: string
  showKUNGalgamePreferOriginalName: boolean
  showKUNGalgameBackground: number
  showKUNGalgameBackgroundBlur: number
  showKUNGalgameBackgroundBrightness: number
  showKUNGalgameBackgroundOpacity: number
  showKUNGalgameBackLoli: boolean
  showKUNGalgameNoResource: boolean
  showKUNGalgameAllOriginalLanguages: boolean
  showKUNGalgameRounded: 'none' | 'sm' | 'md' | 'lg'
  showKUNGalgamePhoneCardColumns: 2 | 3
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
