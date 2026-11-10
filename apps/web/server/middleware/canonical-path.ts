import { canonicalPath } from '../utils/kunLocation'
import { kunRedirect } from '../utils/kunRedirect'

// /topic/1/ rendered the same page as /topic/1 under its own URL, which a
// crawler indexes as a duplicate; /galgame/resource/12342 with a copied
// zero-width suffix 404ed.
export default defineEventHandler((event) => {
  if (event.method !== 'GET' && event.method !== 'HEAD') return
  const q = event.path.indexOf('?')
  const path = q === -1 ? event.path : event.path.slice(0, q)
  if (path.startsWith('/api/') || path.startsWith('/_')) return

  const to = canonicalPath(path)
  if (to === path) return
  // //host (or /\host, which browsers read the same) would be a
  // protocol-relative redirect off the site.
  if (/^\/[/\\]/.test(to)) return
  return kunRedirect(event, q === -1 ? to : to + event.path.slice(q), 301)
})
