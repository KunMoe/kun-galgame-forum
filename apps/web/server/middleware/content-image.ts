import { imageHashUrl, parseImageToken } from '#shared/utils/imageService'

export default defineEventHandler((event) => {
  if (event.method !== 'GET') return
  const path = event.path.split('?')[0]!
  if (!path.startsWith('/image/')) return
  const parsed = parseImageToken(path.slice('/image/'.length))
  if (!parsed) return

  const base = String(useRuntimeConfig(event).public.imageCdnBase || '')
  // Without a cache header every load of a token re-asked kungal.com, and a
  // sticker picker's worth of them tripped Cloudflare's rate limit (1015).
  setResponseHeader(
    event,
    'cache-control',
    'public, max-age=31536000, immutable'
  )
  return sendRedirect(
    event,
    imageHashUrl(base, parsed.hash, parsed.variant),
    302
  )
})
