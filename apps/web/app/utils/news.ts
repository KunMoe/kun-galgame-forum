// A feed page is grouped on (date, source) rather than on date alone; the
// reasoning lives on KunNewsGroup. Both the scrolling feed and the month
// archive cut their lists the same way, so the rule is written once here.
export const groupNewsItems = (
  items: KunNewsItem[],
  sources: Record<string, KunNewsSource>
): KunNewsGroup[] => {
  const out: KunNewsGroup[] = []
  for (const item of items) {
    const date = formatDate(item.published_at, { isShowYear: true })
    const key = `${date}-${item.news_source}`
    const last = out.at(-1)
    if (last?.key === key) {
      last.items.push(item)
      continue
    }
    out.push({ key, date, source: sources[item.news_source], items: [item] })
  }
  return out
}
