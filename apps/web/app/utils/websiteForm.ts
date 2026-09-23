import type { AdminWebsite } from '#shared/utils/api/schemas'
import type { WebsiteForm } from '~/components/website/modal/types'

export const websiteFormOf = (source: AdminWebsite): WebsiteForm => ({
  host: source.host,
  title: source.title,
  description: source.description,
  icon_image_hash: source.icon?.hash ?? '',
  website_category_id: source.website_category_id,
  website_tag_ids: [...source.website_tag_ids],
  is_nsfw: source.is_nsfw,
  state: source.state,
  language: source.language,
  urls: [...source.urls],
  founded: source.founded
})

// PATCH sends only what changed: the edit form used to post every field back,
// and five sites whose stored icon began with a space could never be saved.
export const changedWebsiteFields = (
  before: WebsiteForm,
  after: WebsiteForm
): Partial<WebsiteForm> => {
  const changed: Partial<WebsiteForm> = {}
  for (const key of Object.keys(after) as (keyof WebsiteForm)[]) {
    if (JSON.stringify(after[key]) !== JSON.stringify(before[key])) {
      Object.assign(changed, { [key]: after[key] })
    }
  }
  return changed
}
