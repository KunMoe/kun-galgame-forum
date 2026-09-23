import { settle } from '#shared/utils/api/problem'
import type { Preferences } from '#shared/utils/api/schemas'
import type { KUNGalgameSettingsStore } from '~/store/types/settings'

export interface KunCloudPreferences {
  show_platform: boolean
  show_rating: boolean
  show_view_like: boolean
  show_update_time: boolean
  show_nsfw_badge: boolean
  show_company: boolean
  show_japanese_name: boolean
  is_open_in_new_tab: boolean
  prefer_original_name: boolean
  no_resource: boolean
  phone_columns: number
  rounded: string
  gallery_sexual_levels: number[]
  gallery_violence_levels: number[]
}

const WRITE_DEBOUNCE_MS = 1000

const DEFAULTS: KunCloudPreferences = {
  show_platform: true,
  show_rating: true,
  show_view_like: true,
  show_update_time: true,
  show_nsfw_badge: false,
  show_company: true,
  show_japanese_name: false,
  is_open_in_new_tab: false,
  prefer_original_name: false,
  no_resource: false,
  phone_columns: 2,
  rounded: 'md',
  gallery_sexual_levels: [],
  gallery_violence_levels: []
}

// All of this is module scope on purpose. A debouncer built inside the
// composable is rebuilt on every call, so each keystroke got its own timer and
// nothing was ever coalesced — the moyu conversion shipped that and sent one
// PUT per switch flip.
let cloudVersion = 0
let cloudUnavailable = false
let applyingRemote = false
let syncStarted = false
let watching = false
let writeTimer: ReturnType<typeof setTimeout> | null = null
let inFlight: Promise<void> | null = null

const isSameDoc = (a: KunCloudPreferences, b: KunCloudPreferences) =>
  JSON.stringify(a) === JSON.stringify(b)

export const useCloudPreferences = () => {
  const api = useApiClient()
  const cardStore = usePersistGalgameCardStore()
  const settingsStore = usePersistSettingsStore()

  const snapshot = (): KunCloudPreferences => ({
    show_platform: cardStore.showPlatform,
    show_rating: cardStore.showRating,
    show_view_like: cardStore.showViewLike,
    show_update_time: cardStore.showUpdateTime,
    show_nsfw_badge: cardStore.showNsfwBadge,
    show_company: cardStore.showCompany,
    show_japanese_name: cardStore.showJapaneseName,
    is_open_in_new_tab: cardStore.isOpenInNewTab,
    prefer_original_name: settingsStore.showKUNGalgamePreferOriginalName,
    no_resource: settingsStore.showKUNGalgameNoResource,
    phone_columns: settingsStore.showKUNGalgamePhoneColumns,
    rounded: settingsStore.showKUNGalgameRounded,
    gallery_sexual_levels: [...settingsStore.showKUNGalgameGallerySexualLevels],
    gallery_violence_levels: [
      ...settingsStore.showKUNGalgameGalleryViolenceLevels
    ]
  })

  // The local stores stay cookie-persisted after this: the API still reads
  // 优先显示日语原名 off the KUNGalgameSettings cookie during SSR, so a cloud
  // value that never lands in the cookie would not reach the rendered names.
  const applyDoc = (doc: Partial<KunCloudPreferences>) => {
    applyingRemote = true
    try {
      if (typeof doc.show_platform === 'boolean')
        cardStore.showPlatform = doc.show_platform
      if (typeof doc.show_rating === 'boolean')
        cardStore.showRating = doc.show_rating
      if (typeof doc.show_view_like === 'boolean')
        cardStore.showViewLike = doc.show_view_like
      if (typeof doc.show_update_time === 'boolean')
        cardStore.showUpdateTime = doc.show_update_time
      if (typeof doc.show_nsfw_badge === 'boolean')
        cardStore.showNsfwBadge = doc.show_nsfw_badge
      if (typeof doc.show_company === 'boolean')
        cardStore.showCompany = doc.show_company
      if (typeof doc.show_japanese_name === 'boolean')
        cardStore.showJapaneseName = doc.show_japanese_name
      if (typeof doc.is_open_in_new_tab === 'boolean')
        cardStore.isOpenInNewTab = doc.is_open_in_new_tab
      if (typeof doc.prefer_original_name === 'boolean')
        settingsStore.showKUNGalgamePreferOriginalName = doc.prefer_original_name
      if (typeof doc.no_resource === 'boolean')
        settingsStore.showKUNGalgameNoResource = doc.no_resource
      if (doc.phone_columns === 2 || doc.phone_columns === 3)
        settingsStore.showKUNGalgamePhoneColumns = doc.phone_columns
      if (isRounded(doc.rounded)) settingsStore.setKUNGalgameRounded(doc.rounded)
      if (Array.isArray(doc.gallery_sexual_levels))
        settingsStore.showKUNGalgameGallerySexualLevels =
          doc.gallery_sexual_levels.filter(isLevel)
      if (Array.isArray(doc.gallery_violence_levels))
        settingsStore.showKUNGalgameGalleryViolenceLevels =
          doc.gallery_violence_levels.filter(isLevel)
    } finally {
      applyingRemote = false
    }
  }

  const pull = async (): Promise<Preferences | null> => {
    const result = await settle(api.GET('/me/preferences'))
    if (!result.ok) {
      if (result.problem.code === 'SCOPE_REQUIRED') {
        cloudUnavailable = true
        return null
      }
      reportProblem(result.problem)
      return null
    }
    return result.data
  }

  const push = async (): Promise<void> => {
    if (cloudUnavailable) return
    const result = await settle(
      api.PUT('/me/preferences', {
        params: {
          header: { 'If-Match': `"${cloudVersion}"` }
        },
        body: { doc: snapshot() as unknown as Preferences['doc'] }
      })
    )
    if (result.ok) {
      cloudVersion = result.data.version
      return
    }
    if (result.problem.code === 'SCOPE_REQUIRED') {
      cloudUnavailable = true
      return
    }
    if (result.problem.status === 412) {
      const fresh = await pull()
      if (fresh) {
        cloudVersion = fresh.version
        applyDoc(fresh.doc as Partial<KunCloudPreferences>)
      }
      return
    }
    reportProblem(result.problem)
  }

  const scheduleWrite = () => {
    if (cloudUnavailable) return
    if (writeTimer) clearTimeout(writeTimer)
    writeTimer = setTimeout(() => {
      writeTimer = null
      inFlight = push().finally(() => {
        inFlight = null
      })
    }, WRITE_DEBOUNCE_MS)
  }

  const flush = async (): Promise<void> => {
    if (writeTimer) {
      clearTimeout(writeTimer)
      writeTimer = null
      inFlight = push().finally(() => {
        inFlight = null
      })
    }
    if (inFlight) await inFlight
  }

  const startWatching = () => {
    if (watching) return
    watching = true
    watch(
      snapshot,
      () => {
        if (!applyingRemote) scheduleWrite()
      },
      { deep: true }
    )
  }

  const sync = async (): Promise<void> => {
    if (syncStarted || cloudUnavailable || !import.meta.client) return
    syncStarted = true

    const resp = await pull()
    if (!resp) return

    cloudVersion = resp.version
    if (resp.version === 0) {
      if (!isSameDoc(snapshot(), DEFAULTS)) await push()
    } else {
      applyDoc(resp.doc as Partial<KunCloudPreferences>)
    }
    startWatching()
  }

  return { sync, flush }
}

const isLevel = (n: unknown): n is number =>
  n === 1 || n === 2 || n === 3

const isRounded = (
  value: unknown
): value is KUNGalgameSettingsStore['showKUNGalgameRounded'] =>
  value === 'none' || value === 'sm' || value === 'md' || value === 'lg'
