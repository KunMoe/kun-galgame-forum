import type { GalgameArtMeta } from '../types/galgame'

export interface KunArtBox {
  width: number
  height: number
}

// Character art is small — a standing figure is ~470px tall, a bust 256×300 —
// and boxes sized from the layout alone drew it 1.3–1.4× past its pixels
// (2026-09-25). The box fits inside max and never grows past the image.
export const artBox = (
  meta: GalgameArtMeta | undefined,
  maxWidth: number,
  maxHeight: number
): KunArtBox => {
  if (!meta || meta.width <= 0 || meta.height <= 0) {
    return { width: maxWidth, height: maxHeight }
  }
  const scale = Math.min(1, maxWidth / meta.width, maxHeight / meta.height)
  return {
    width: Math.round(meta.width * scale),
    height: Math.round(meta.height * scale)
  }
}
