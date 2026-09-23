import type {
  AspectScores,
  Rating,
  RatingSummary
} from '#shared/utils/api/schemas'
import type { CatalogName } from '#shared/utils/catalogName'
import type { KunGalgameDim } from '~/constants/galgame-rating'

type NameOf = (n: CatalogName) => { name: string; original: string }

export const aspectDims = (s: AspectScores): Record<KunGalgameDim, number> => ({
  art: s.art ?? 0,
  story: s.story ?? 0,
  music: s.music ?? 0,
  character: s.character ?? 0,
  route: s.route ?? 0,
  system: s.system ?? 0,
  voice: s.voice ?? 0,
  replay_value: s.replay_value ?? 0
})

export const dimsToAspectScores = (
  d: Record<KunGalgameDim, number>
): AspectScores => ({
  art: d.art || null,
  story: d.story || null,
  music: d.music || null,
  character: d.character || null,
  route: d.route || null,
  system: d.system || null,
  voice: d.voice || null,
  replay_value: d.replay_value || null
})

// The rating card on the user pages and the work page still renders the legacy
// shape; the rating faces read v1, so they translate at the edge.
export const ratingToCard = (
  r: RatingSummary,
  nameOf: NameOf
): GalgameRatingCard => ({
  id: Number(r.id),
  user: toKunUser(r.author),
  recommend: r.recommend,
  overall: r.overall,
  view: r.view_count,
  galgame_type: [...r.game_types],
  play_status: r.play_status,
  short_summary: r.short_summary,
  ...aspectDims(r.aspect_scores),
  spoiler_level: r.spoiler_level,
  like_count: r.like_count,
  created: r.created_at,
  updated: r.updated_at,
  galgame: {
    id: Number(r.work?.id ?? 0),
    name: r.work ? nameOf(r.work).name : '',
    content_limit: r.work?.is_nsfw ? 'nsfw' : 'sfw'
  }
})

export const ratingToGalgamePageCard = (
  r: Rating,
  nameOf: NameOf
): GalgameRatingCardOnGalgamePage => ({
  ...ratingToCard(r, nameOf),
  is_liked: r.viewer?.has_liked ?? false
})
