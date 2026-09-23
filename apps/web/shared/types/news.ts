import type { components } from './api/v1'

export type KunNewsItem = components['schemas']['NewsItem']
export type KunNewsSource = components['schemas']['NewsSource']
export type KunNewsArchive = components['schemas']['NewsArchive']
export type KunNewsArchiveMonth = KunNewsArchive['months'][number]

// A feed page is grouped on (date, source), never on date alone: one partner
// republishes a whole week of bulletins under a single timestamp, and a header
// that spanned two partners would print one partner's attribution over the
// other's items.
export interface KunNewsGroup {
  key: string
  date: string
  source: KunNewsSource | undefined
  items: KunNewsItem[]
}
