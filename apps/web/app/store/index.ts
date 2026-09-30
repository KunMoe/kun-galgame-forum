import { usePersistEditGalgameRatingStore } from './modules/edit/rating'

export const createEmptyLocaleMap = () => ({
  'en-us': '',
  'ja-jp': '',
  'zh-cn': '',
  'zh-tw': ''
})

export const resetReactiveState = <T extends Record<string, unknown>>(
  target: T,
  defaults: T
) => {
  Object.assign(target, defaults)
}

export const kungalgameStoreReset = () => {
  const { resetEditGalgameStore } = usePersistEditGalgameStore()
  resetEditGalgameStore()

  const persistEditTopicStore = usePersistEditTopicStore()
  persistEditTopicStore.mode = 'preview'
  persistEditTopicStore.resetTopicData()

  const persistEditGalgameRatingStore = usePersistEditGalgameRatingStore()
  persistEditGalgameRatingStore.shortSummary = ''

  const persistReplyStore = usePersistKUNGalgameReplyStore()
  persistReplyStore.mode = 'preview'
  persistReplyStore.resetReplyDraft()

  const { resetUser } = usePersistUserStore()
  resetUser()

  const persistSettingsStore = usePersistSettingsStore()
  persistSettingsStore.showKUNGalgamePageTransparency = 80
  persistSettingsStore.showKUNGalgameContentLimit = 'sfw'
  persistSettingsStore.showKUNGalgamePreferOriginalName = false
  persistSettingsStore.showKUNGalgameBackground = 0
  persistSettingsStore.showKUNGalgameBackgroundBlur = 0
  persistSettingsStore.showKUNGalgameBackgroundBrightness = 100
  persistSettingsStore.showKUNGalgameBackLoli = false

  usePersistGalgameCardStore().reset()

  const tempEditStore = useTempEditStore()
  tempEditStore.resetRewriteTopicData()

  const tempGalgamePRStore = useTempGalgamePRStore()
  tempGalgamePRStore.galgamePR = []

  const tempReplyStore = useTempReplyStore()
  tempReplyStore.isEdit = false
  tempReplyStore.isScrollToTop = false
  tempReplyStore.scrollToReplyId = -1
  tempReplyStore.isReplyRewriting = false
  tempReplyStore.replyRewrite = null
  tempReplyStore.lastSuccessfulReply = null

  const componentMessageStore = useComponentMessageStore()
  componentMessageStore.showAlert = false
  componentMessageStore.alertTitle = ''
  componentMessageStore.alertMsg = ''
  componentMessageStore.isShowCancel = false
  componentMessageStore.isShowCapture = false
  componentMessageStore.isCaptureSuccessful = false
  componentMessageStore.codeSalt = ''

  const tempSettingStore = useTempSettingStore()
  tempSettingStore.reset()
  tempSettingStore.messageStatus = 'offline'
}
