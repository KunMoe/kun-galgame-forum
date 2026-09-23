import { describe, expect, it } from 'vitest'
import {
  KUN_GALGAME_EXTERNAL_RATING_MAP,
  KUN_GALGAME_LOCAL_RATING_META
} from '~/constants/galgame-rating'
import { externalRatingHistogram } from './_stats'

const axisOf = (source: 'vndb' | 'bangumi' | 'erogamescape' | 'dlsite') =>
  KUN_GALGAME_EXTERNAL_RATING_MAP[source].histogram

describe('externalRatingHistogram', () => {
  it('fills the buckets nobody voted for back in', () => {
    const buckets = externalRatingHistogram(
      [
        { bucket: 3, vote_count: 1 },
        { bucket: 10, vote_count: 4 }
      ],
      axisOf('bangumi').keys
    )
    expect(buckets).toEqual([0, 0, 1, 0, 0, 0, 0, 0, 0, 4])
  })

  it('lands 批评空间 deciles on eleven slots, not ninety empty columns', () => {
    const axis = axisOf('erogamescape')
    const buckets = externalRatingHistogram(
      [
        { bucket: 0, vote_count: 2 },
        { bucket: 70, vote_count: 9 },
        { bucket: 100, vote_count: 1 }
      ],
      axis.keys
    )
    expect(buckets).toHaveLength(11)
    expect(buckets[0]).toBe(2)
    expect(buckets[7]).toBe(9)
    expect(buckets[10]).toBe(1)
    expect(axis.keys.map(axis.label)).toEqual([
      '0-9',
      '10-19',
      '20-29',
      '30-39',
      '40-49',
      '50-59',
      '60-69',
      '70-79',
      '80-89',
      '90-99',
      '100'
    ])
  })

  it('reads a key as the key, never as an offset', () => {
    const [first] = externalRatingHistogram(
      [{ bucket: 1, vote_count: 7 }],
      axisOf('dlsite').keys
    )
    expect(first).toBe(7)
    expect(
      externalRatingHistogram([{ bucket: 0, vote_count: 7 }], [1, 2, 3])
    ).toEqual([0, 0, 0])
  })

  it('gives every source an axis long enough for its own scale', () => {
    expect(axisOf('vndb').keys).toHaveLength(10)
    expect(axisOf('dlsite').keys).toHaveLength(5)
    expect(KUN_GALGAME_LOCAL_RATING_META.histogram.keys).toHaveLength(10)
  })
})
