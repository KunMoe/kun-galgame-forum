import type { StickerPack } from '@kungal/editor-core'
import { settle } from '#shared/utils/api/problem'
import { imageHashUrl } from '#shared/utils/imageService'
import { imageCdnBase } from '~/utils/imageSrc'

const stickerHash = (src: string) => src.match(/[0-9a-f]{64}/)?.[0]

/**
 * The official sticker packs, fetched once per page and shared by every
 * picker (the editor's and the private-message composer's).
 *
 * `useState` rather than a module-level ref: a module ref is shared across SSR
 * requests on the server, and this list is fetched lazily so it would be
 * whatever the first request happened to get.
 */
export const useStickerPacks = () => {
  const packs = useState<StickerPack[]>('kun-sticker-packs', () => [])

  const load = async (): Promise<StickerPack[]> => {
    if (packs.value.length) {
      return packs.value
    }
    const result = await settle(useApiClient().GET('/sticker-packs'))
    if (!result.ok) {
      throw new Error('sticker packs unavailable')
    }
    // The picker loads and inserts the same src. It must be the CDN itself:
    // an /image/<hash> token here made every sticker a redirect through
    // kungal.com, and opening the picker tripped Cloudflare's rate limit
    // (1015). The API folds this URL back into the token on every write.
    const base = imageCdnBase()
    packs.value = result.data.items.map((pack) => ({
      name: pack.display_name,
      stickers: pack.stickers.flatMap((sticker) =>
        sticker.image
          ? [
              {
                name: sticker.display_name,
                src: imageHashUrl(base, sticker.image.hash, '320')
              }
            ]
          : []
      )
    }))
    return packs.value
  }

  const hashes = computed(
    () =>
      new Set(
        packs.value.flatMap((pack) =>
          pack.stickers.flatMap((sticker) => stickerHash(sticker.src) ?? [])
        )
      )
  )

  const isSticker = (src: string) => {
    const hash = stickerHash(src)
    return !!hash && hashes.value.has(hash)
  }

  return { packs, load, isSticker }
}
