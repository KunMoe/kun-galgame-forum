// /topic/1/ rendered the same page as /topic/1 under its own URL, which a
// crawler indexes as a duplicate.
export default defineEventHandler((event) => {
  if (event.method !== 'GET' && event.method !== 'HEAD') return
  const [path, query] = event.path.split('?') as [string, string?]
  if (path.length <= 1 || !path.endsWith('/')) return
  // //host/ (or /\host/, which browsers read the same) would become a
  // protocol-relative redirect off the site.
  if (/^\/[/\\]/.test(path)) return
  if (path.startsWith('/api/') || path.startsWith('/_')) return
  const to = path.replace(/\/+$/, '') || '/'
  return sendRedirect(event, query ? `${to}?${query}` : to, 301)
})
