const REF_RE = /^([0-9a-f]{64})(?:_([a-z0-9]+))?$/
const ABS_RE =
  /^https?:\/\/([A-Za-z0-9.-]+)\/([0-9a-f]{2})\/([0-9a-f]{2})\/([0-9a-f]{64})(_[a-z0-9]+)?\.webp$/

const IMAGE_SERVICE_HOSTS = new Set([
  'image.kungal.iloveren.link',
  'image.kungal.com'
])

export type ParsedImageRef = {
  hash: string
  variant?: string
}

export const parseImageRef = (ref: string): ParsedImageRef | null => {
  const m = ref.match(REF_RE)
  if (!m) return null
  return m[2] ? { hash: m[1]!, variant: m[2] } : { hash: m[1]! }
}

export const imageRefCdnUrl = (
  base: string,
  parsed: ParsedImageRef
): string => {
  const b = (base || '').replace(/\/+$/, '')
  const file = parsed.variant
    ? `${parsed.hash}_${parsed.variant}.webp`
    : `${parsed.hash}.webp`
  return `${b}/${parsed.hash.slice(0, 2)}/${parsed.hash.slice(2, 4)}/${file}`
}

export const hostFromCdnBase = (base: string): string => {
  try {
    return new URL(base).host.toLowerCase()
  } catch {
    return ''
  }
}

export const parseAbsoluteImageUrl = (
  src: string,
  extraHost = ''
): ParsedImageRef | null => {
  const m = src.match(ABS_RE)
  if (!m) return null
  const host = m[1]!.toLowerCase()
  if (!IMAGE_SERVICE_HOSTS.has(host) && host !== extraHost.toLowerCase()) {
    return null
  }
  const aa = m[2]!
  const bb = m[3]!
  const hash = m[4]!
  if (aa !== hash.slice(0, 2) || bb !== hash.slice(2, 4)) return null
  return m[5] ? { hash, variant: m[5].slice(1) } : { hash }
}

export type StickerItem = {
  src: string
  name: string
  hash?: string
}

export type StickerPack = {
  name: string
  stickers: StickerItem[]
}

export type StickerPacksPayload = {
  packs: StickerPack[]
  variant?: string
}

export const transformStickerPacks = (
  data: StickerPacksPayload,
  extraHost = ''
): { packs: StickerPack[] } => {
  const variant = data.variant || '320'
  return {
    packs: data.packs.map((pack) => ({
      ...pack,
      stickers: pack.stickers.map((sticker) => {
        const fromHash = sticker.hash ? parseImageRef(sticker.hash) : null
        const parsed = fromHash ?? parseAbsoluteImageUrl(sticker.src, extraHost)
        if (!parsed) return sticker
        return { ...sticker, src: `/image/${parsed.hash}_${variant}` }
      })
    }))
  }
}
