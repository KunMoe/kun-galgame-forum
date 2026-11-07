// The image service is shared by every NextMoe site; only its public domain may
// differ. Production sets the domain once, in docker-compose.prod.yml, for both
// the API and the web; this default only covers dev and tests.
export const DEFAULT_IMAGE_CDN_BASE = 'https://image.kungal.iloveren.link'

const TOKEN_RE = /^([0-9a-f]{64})(?:_([a-z0-9]+))?$/

export const imageHashUrl = (
  base: string,
  hash: string,
  variant?: string
): string => {
  const b = (base || '').replace(/\/+$/, '')
  const file = variant ? `${hash}_${variant}.webp` : `${hash}.webp`
  return `${b}/${hash.slice(0, 2)}/${hash.slice(2, 4)}/${file}`
}

export const parseImageToken = (
  ref: string
): { hash: string; variant?: string } | null => {
  const m = ref.match(TOKEN_RE)
  if (!m) return null
  return m[2] ? { hash: m[1]!, variant: m[2] } : { hash: m[1]! }
}
