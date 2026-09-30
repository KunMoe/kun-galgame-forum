import type { components, operations } from './api/v1'

export type KunNewsItem = components['schemas']['NewsItem']
export type KunNewsSource = components['schemas']['NewsSource']
export type KunNewsArchive = components['schemas']['NewsArchive']
export type KunNewsArchiveMonth = KunNewsArchive['months'][number]
export type KunNewsItemDetail = components['schemas']['NewsItemDetail']
export type KunNewsSubmission = components['schemas']['NewsSubmission']
export type KunNewsSubmissionState = KunNewsSubmission['state']
export type KunNewsSubmissionCreate =
  operations['createNewsSubmission']['requestBody']['content']['application/json']
export type KunNewsSubmissionPatch =
  operations['updateNewsSubmission']['requestBody']['content']['application/json']

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
