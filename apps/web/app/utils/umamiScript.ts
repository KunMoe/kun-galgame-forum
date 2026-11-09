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
  if (!website || !origin || !URL.canParse(origin)) return ''
  return `${origin}/script.js`
}
