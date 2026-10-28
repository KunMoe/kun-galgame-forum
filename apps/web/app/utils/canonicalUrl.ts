import type { RouteLocationNormalizedLoaded } from 'vue-router'

// A list's later pages point at themselves, not at page 1: canonicalizing them
// away hid every item past the first page from a crawler. No other query
// makes a different document.
export const kunCanonicalUrl = (
  route: Pick<RouteLocationNormalizedLoaded, 'path' | 'query'>
) => {
  const path = route.path === '/' ? '/' : route.path.replace(/\/+$/, '')
  const page = Number(route.query.page)
  // A tab's own ?page= (the work page's quiz panel) is not a page of the URL.
  const paged = Number.isInteger(page) && page > 1 && !route.query.tab
  const query = paged ? `?page=${page}` : ''
  return `${kungal.domain.main}${path}${query}`
}
