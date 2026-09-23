import {
  KUN_GALGAME_DIMENSIONS,
  type KunGalgameDim
} from '~/constants/galgame-rating'
import type { WorkExternalRatingBucket } from '#shared/utils/api/schemas'
import type { GalgameRatingCardOnGalgamePage } from '~~/shared/types/galgame-rating'

export const ratingMean = (values: number[]) =>
  values.length ? values.reduce((sum, v) => sum + v, 0) / values.length : 0

export const ratingHistogram = (
  ratings: GalgameRatingCardOnGalgamePage[],
  max = 10
) => {
  const buckets = Array.from({ length: max }, () => 0)
  for (const rating of ratings) {
    const index = Math.round(rating.overall) - 1
    if (index >= 0 && index < max) {
      buckets[index]! += 1
    }
  }
  return buckets
}

// Catalog ships the histogram sparsely: only the values someone actually voted
// for are present, so an absent bucket is a real zero and has to be filled back
// in or the columns silently shift left. The keys are not one shared axis
// either — 批评空间's are deciles (0, 10, … 100), everyone else's are points.
export const externalRatingHistogram = (
  distribution: WorkExternalRatingBucket[],
  keys: number[]
) => {
  const slot = new Map(keys.map((key, index) => [key, index]))
  const buckets = Array.from({ length: keys.length }, () => 0)
  for (const bucket of distribution) {
    const index = slot.get(bucket.bucket)
    if (index !== undefined) {
      buckets[index]! += bucket.vote_count
    }
  }
  return buckets
}

export const ratingDimensionMeans = (
  ratings: GalgameRatingCardOnGalgamePage[]
) =>
  Object.fromEntries(
    KUN_GALGAME_DIMENSIONS.map((dim) => [
      dim,
      ratingMean(ratings.map((rating) => rating[dim]))
    ])
  ) as Record<KunGalgameDim, number>

export const ratingTally = (values: string[]) => {
  const tally = new Map<string, number>()
  for (const value of values) {
    tally.set(value, (tally.get(value) ?? 0) + 1)
  }
  return [...tally].sort((a, b) => b[1] - a[1])
}
