import type { WorkSummary } from '#shared/utils/api/schemas'
import { GALGAME_RESOURCE_PLATFORM_ICON_MAP } from '~/constants/galgameResource'
import type { CatalogName } from '#shared/utils/catalogName'

type NameOf = (n: CatalogName) => { name: string; original: string }

// The galgame card still renders the legacy list shape that /galgame sends;
// the entity pages read v1, so they translate at the edge until the card moves.
export const workSummaryToCard = (
  w: WorkSummary,
  nameOf: NameOf
): GalgameCard => {
  const { name, original } = nameOf(w)
  const maker = w.maker ? nameOf(w.maker).name : ''
  return {
    id: Number(w.id),
    name,
    name_original: original,
    user: { id: 0, name: '', avatar: '' },
    content_limit: w.is_nsfw ? 'nsfw' : 'sfw',
    view: w.view_count,
    like_count: w.like_count,
    rating: w.rating_score ?? 0,
    rating_count: w.rating_count,
    is_on_forum: w.is_published,
    platform: w.resource_platforms.filter(
      (p) => p in GALGAME_RESOURCE_PLATFORM_ICON_MAP
    ),
    language: [...w.resource_languages],
    resource_update_time: w.resource_updated_at ?? '',
    release_date: w.release_date,
    release_precision: w.release_date_precision ?? 'unknown',
    effective_banner_hash: w.banner?.hash,
    effective_banner_url: w.banner?.url,
    effective_banner_width: w.banner?.width ?? undefined,
    effective_banner_height: w.banner?.height ?? undefined,
    effective_banner_thumbhash: w.banner?.thumbhash ?? undefined,
    effective_portrait_hash: w.cover?.hash,
    effective_portrait_url: w.cover?.url,
    effective_portrait_width: w.cover?.width ?? undefined,
    effective_portrait_height: w.cover?.height ?? undefined,
    effective_portrait_thumbhash: w.cover?.thumbhash ?? undefined,
    company: maker
  }
}
