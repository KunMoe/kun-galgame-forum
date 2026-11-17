// The library answers popularity and release date only; a classic ranking
// opens on its default, popularity.
const LIBRARY_SORTS: Record<string, string> = {
  newest: 'release_date'
}

const SORT_LABELS: Record<string, string> = {
  popular: '热门',
  newest: '最新'
}

const MAX_TAGS = 10

export const agentBrowseLink = (b: AgentBrowse) => {
  if (b.target !== 'works') return null
  const ids = b.ids.filter((id) => /^[1-9]\d*$/.test(id)).slice(0, MAX_TAGS)
  if (!ids.length) return null
  const tags = ids.join(',')
  const names = b.names.map((n) => `「${n}」`).join('')
  const sp = new URLSearchParams({ tags })
  const sort = LIBRARY_SORTS[b.sort]
  if (sort) sp.set('sortField', sort)
  return {
    key: `${tags}:${b.sort}`,
    label: `去 Galgame 数据库按${names || '标签'}筛选${SORT_LABELS[b.sort] ?? ''}作品`,
    href: `/gallib?${sp.toString()}`
  }
}
