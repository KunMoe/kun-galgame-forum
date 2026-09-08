import type { StickerPack } from '@kungal/editor-core'

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
    const res = await $fetch<{ packs: StickerPack[] }>('/api/sticker-packs')
    packs.value = res.packs
    return packs.value
  }

  return { packs, load }
}
