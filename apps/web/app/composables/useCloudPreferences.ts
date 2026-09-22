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

interface KunPreferencesEnvelope {
  // The proxy hands back whatever the account holds, and a namespace nobody has
  // written yet answers 200 with an empty document rather than 404.
  doc: Partial<KunCloudPreferences> | null
  version: number
  updated_at: string | null
}

const CODE_CLOUD_PREFERENCES_UNAVAILABLE = 240
const CODE_CLOUD_PREFERENCES_CONFLICT = 241

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

  const push = async (): Promise<void> => {
    if (cloudUnavailable) return
    let conflicted = false
    const resp = await kunFetch<KunPreferencesEnvelope>('/user/preferences', {
      method: 'PUT',
      body: { doc: snapshot() },
      headers: { 'If-Match': `"${cloudVersion}"` },
      onApiError: (envelope: { code: number; message: string }) => {
        if (envelope.code === CODE_CLOUD_PREFERENCES_UNAVAILABLE) {
          cloudUnavailable = true
          return true
        }
        if (envelope.code === CODE_CLOUD_PREFERENCES_CONFLICT) {
          conflicted = true
          return true
        }
        return false
      }
    })
    if (resp) {
      cloudVersion = resp.version
      return
    }
    if (conflicted) {
      const fresh = await pull()
      if (fresh) {
        cloudVersion = fresh.version
        applyDoc(fresh.doc ?? {})
      }
    }
  }

  const pull = async (): Promise<KunPreferencesEnvelope | null> =>
    kunFetch<KunPreferencesEnvelope>('/user/preferences', {
      onApiError: (envelope: { code: number; message: string }) => {
        if (envelope.code === CODE_CLOUD_PREFERENCES_UNAVAILABLE) {
          cloudUnavailable = true
          return true
        }
        return false
      }
    })

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
      applyDoc(resp.doc ?? {})
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
