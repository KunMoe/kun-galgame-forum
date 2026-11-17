// URL.canParse needs Chrome 120; OPPO's built-in browser lacks it, and the
// throw inside app.vue's computed turned every page into a 500.
const isUrl = (value: string) => {
  try {
    new URL(value)
    return true
  } catch {
    return false
  }
}

export const umamiTrackerSrc = (
  host: unknown,
  id: unknown,
  dev = process.env.NODE_ENV === 'development'
) => {
  if (dev) return ''
  const website = String(id ?? '').trim()
  const origin = String(host ?? '')
    .trim()
    .replace(/\/+$/, '')
  if (!website || !origin || !isUrl(origin)) return ''
  return `${origin}/script.js`
}
