interface StickerItem {
  src: string
  name: string
}

interface StickerPack {
  name: string
  stickers: StickerItem[]
}

/**
 * The official sticker packs, proxied from sticker.kungal.com.
 *
 * This used to be `app/constants/sticker.ts`: 498 URLs generated in the
 * browser from `sticker.kungal.com/stickers/KUNgal{set}/{n}.webp` and a
 * hardcoded `[80,80,80,80,80,80,18]`. Every one of them 404s today — that
 * path addressed a position in a collection on a site we do not own, and the
 * site stopped serving static files. The array was also wrong whenever a pack
 * gained a sticker, which nobody noticed because nobody could notice.
 *
 * Server-side because the browser must not reach across origins for this, and
 * cached because 82 KB is worth fetching once an hour rather than per picker
 * open. `staleMaxAge` is the load-bearing part: if the sticker site is down we
 * keep serving last week's packs instead of an empty picker.
 *
 * NOT under `/api/`, which on this site is Traefik's: it routes
 * kungal.com/api/* to the Go backend, so a Nitro handler placed there is
 * unreachable from a browser and answers the Go API's 401 envelope instead.
 * (`server/api/__sitemap__/urls.ts` gets away with it only because the sitemap
 * module calls it inside Nitro, never over the network.)
 */
export default defineCachedEventHandler(
  async (): Promise<{ packs: StickerPack[] }> => {
    const base = useRuntimeConfig().stickerBaseUrl
    const res = await $fetch<{
      code: number
      message: string
      data: { packs: StickerPack[] } | null
    }>(`${base}/api/v1/editor-packs`, { timeout: 8000 })
    if (res.code !== 0 || !res.data) {
      throw createError({
        statusCode: 502,
        statusMessage: res.message || 'sticker packs unavailable'
      })
    }
    return res.data
  },
  {
    name: 'sticker-packs',
    maxAge: 3600,
    staleMaxAge: 604800,
    swr: true
  }
)
