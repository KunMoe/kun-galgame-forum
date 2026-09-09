import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  ENABLE_KUN_VISUAL_NOVEL_FORUM_WINTER_THEME,
  KUN_VISUAL_NOVEL_FORUM_WINTER_THEME_BACKGROUND
} from '~/config/theme'
import {
  KUN_DEFAULT_FEED_TABS,
  KUN_FEED_TABS_VERSION
} from '~/constants/activity'
import type { KUNGalgameSettingsStore } from '../types/settings'

const SETTINGS_CUSTOM_BACKGROUND_IMAGE_NAME: string = 'kun-galgame-custom-bg'
const SETTINGS_PUBLISH_Banner_IMAGE_NAME: string = 'kun-galgame-publish-banner'
const SETTINGS_DEFAULT_FONT_FAMILY: string = 'system-ui'

export const usePersistSettingsStore = defineStore(
  'KUNGalgameSettings',
  () => {
    const showKUNGalgamePageTransparency =
      ref<KUNGalgameSettingsStore['showKUNGalgamePageTransparency']>(50)
    const showKUNGalgameFontStyle = ref<
      KUNGalgameSettingsStore['showKUNGalgameFontStyle']
    >(SETTINGS_DEFAULT_FONT_FAMILY)
    const showKUNGalgameContentLimit =
      ref<KUNGalgameSettingsStore['showKUNGalgameContentLimit']>('sfw')
    // Read by the API off this store's cookie, not sent as a field: every name
    // the site shows is elected server-side, so the switch has to reach the
    // request that renders them.
    const showKUNGalgamePreferOriginalName =
      ref<KUNGalgameSettingsStore['showKUNGalgamePreferOriginalName']>(false)
    const showKUNGalgameBackground =
      ref<KUNGalgameSettingsStore['showKUNGalgameBackground']>(0)
    const showKUNGalgameBackgroundBlur =
      ref<KUNGalgameSettingsStore['showKUNGalgameBackgroundBlur']>(0)
    const showKUNGalgameBackgroundBrightness =
      ref<KUNGalgameSettingsStore['showKUNGalgameBackgroundBrightness']>(100)
    const showKUNGalgameBackgroundOpacity =
      ref<KUNGalgameSettingsStore['showKUNGalgameBackgroundOpacity']>(15)
    const showKUNGalgameBackLoli =
      ref<KUNGalgameSettingsStore['showKUNGalgameBackLoli']>(false)
    const showKUNGalgameNoResource =
      ref<KUNGalgameSettingsStore['showKUNGalgameNoResource']>(false)
    const showKUNGalgameRounded =
      ref<KUNGalgameSettingsStore['showKUNGalgameRounded']>('md')
    const showKUNGalgamePhoneColumns =
      ref<KUNGalgameSettingsStore['showKUNGalgamePhoneColumns']>(2)
    const showKUNGalgameGallerySexualLevels = ref<
      KUNGalgameSettingsStore['showKUNGalgameGallerySexualLevels']
    >([])
    const showKUNGalgameGalleryViolenceLevels = ref<
      KUNGalgameSettingsStore['showKUNGalgameGalleryViolenceLevels']
    >([])
    const feedTabs = ref<KUNGalgameSettingsStore['feedTabs']>(
      structuredClone(KUN_DEFAULT_FEED_TABS)
    )
    const feedTabsVersion = ref<KUNGalgameSettingsStore['feedTabsVersion']>(0)

    const resetKUNGalgameFeedTabs = () => {
      feedTabs.value = structuredClone(KUN_DEFAULT_FEED_TABS)
      feedTabsVersion.value = KUN_FEED_TABS_VERSION
    }

    const setKUNGalgameFontStyle = (font: string) => {
      showKUNGalgameFontStyle.value = font
      document.documentElement.style.setProperty('--font-family', font)
    }

    const setKUNGalgameTransparency = (trans: number) => {
      showKUNGalgamePageTransparency.value = trans
      const opacity = `${trans / 100}`
      document.documentElement.style.setProperty(
        '--kun-global-opacity',
        opacity
      )
      document.documentElement.style.setProperty(
        '--kun-surface-opacity',
        opacity
      )
    }

    // Two custom properties, one slider. `--kun-background-blur` is this app's
    // own: Sidebar.vue reads it through a `backdrop-blur-[var(...)]` arbitrary
    // value. `--kun-backdrop-filter` is KunUI's, which REPLACED
    // `--kun-background-blur` in @kungal/ui-tokens 1.9.3 -- until it was set
    // here the slider moved nothing on any KunCard, KunModal or floating
    // panel, because KunUI stopped reading the old name entirely.
    //
    // Zero maps to `none`, not `blur(0px)`: a real backdrop-filter promotes
    // every raised surface to its own compositing layer, which is exactly the
    // mobile scroll jank 1.9.3 removed by making the blur opt-in.
    const setKUNGalgameBackgroundBlur = (blur: number) => {
      showKUNGalgameBackgroundBlur.value = blur
      document.documentElement.style.setProperty(
        '--kun-background-blur',
        `${blur}px`
      )
      document.documentElement.style.setProperty(
        '--kun-backdrop-filter',
        blur > 0 ? `blur(${blur}px)` : 'none'
      )
    }

    const setKUNGalgameBackgroundBrightness = (brightness: number) => {
      showKUNGalgameBackgroundBrightness.value = brightness
      document.documentElement.style.setProperty(
        '--kun-background-brightness',
        `${brightness}%`
      )
    }

    const ROUNDED_SCALE: Record<
      KUNGalgameSettingsStore['showKUNGalgameRounded'],
      number
    > = { none: 0, sm: 0.5, md: 1, lg: 1.5 }

    const setKUNGalgameRounded = (
      level: KUNGalgameSettingsStore['showKUNGalgameRounded']
    ) => {
      showKUNGalgameRounded.value = level
      document.documentElement.style.setProperty(
        '--kun-radius-scale',
        `${ROUNDED_SCALE[level]}`
      )
    }

    const setSystemBackground = async (index: number) => {
      showKUNGalgameBackground.value = index
      await deleteImage(SETTINGS_CUSTOM_BACKGROUND_IMAGE_NAME)
    }

    const setCustomBackground = async (file: File) => {
      await saveImage(file, SETTINGS_CUSTOM_BACKGROUND_IMAGE_NAME)
      showKUNGalgameBackground.value = -1
    }

    const getCurrentBackground = async () => {
      const backgroundImageBlobData = await getImage(
        SETTINGS_CUSTOM_BACKGROUND_IMAGE_NAME
      )
      if (showKUNGalgameBackground.value === 0) {
        return ENABLE_KUN_VISUAL_NOVEL_FORUM_WINTER_THEME
          ? KUN_VISUAL_NOVEL_FORUM_WINTER_THEME_BACKGROUND
          : ''
      }

      if (showKUNGalgameBackground.value === -1 && backgroundImageBlobData) {
        return URL.createObjectURL(backgroundImageBlobData)
      }

      return `/bg/bg${showKUNGalgameBackground.value}.webp`
    }

    const setKUNGalgameSettingsRecover = async () => {
      kungalgameStoreReset()
      await deleteImage(SETTINGS_CUSTOM_BACKGROUND_IMAGE_NAME)
      await deleteImage(SETTINGS_PUBLISH_Banner_IMAGE_NAME)
    }

    return {
      showKUNGalgamePageTransparency,
      showKUNGalgameFontStyle,
      showKUNGalgameContentLimit,
      showKUNGalgamePreferOriginalName,
      showKUNGalgameBackground,
      showKUNGalgameBackgroundBlur,
      showKUNGalgameBackgroundBrightness,
      showKUNGalgameBackgroundOpacity,
      showKUNGalgameBackLoli,
      showKUNGalgameNoResource,
      showKUNGalgameRounded,
      showKUNGalgamePhoneColumns,
      showKUNGalgameGallerySexualLevels,
      showKUNGalgameGalleryViolenceLevels,
      feedTabs,
      feedTabsVersion,
      resetKUNGalgameFeedTabs,
      setKUNGalgameFontStyle,
      setKUNGalgameTransparency,
      setKUNGalgameBackgroundBlur,
      setKUNGalgameBackgroundBrightness,
      setKUNGalgameRounded,
      setSystemBackground,
      setCustomBackground,
      getCurrentBackground,
      setKUNGalgameSettingsRecover
    }
  },
  {
    persist: {
      afterHydrate: (ctx) => {
        const store = ctx.store as unknown as {
          feedTabsVersion: number
          feedTabs: KUNGalgameSettingsStore['feedTabs']
        }
        if (store.feedTabsVersion < KUN_FEED_TABS_VERSION) {
          store.feedTabs = structuredClone(KUN_DEFAULT_FEED_TABS)
          store.feedTabsVersion = KUN_FEED_TABS_VERSION
        }
      }
    }
  }
)
