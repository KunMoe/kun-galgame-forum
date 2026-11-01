import type { SiteRating } from '#shared/utils/api/schemas'

export const formatDeviation = (value: number) => value.toFixed(1)

export const formatSiteScore = (value: number, scaleMax: number) =>
  scaleMax >= 100 ? value.toFixed(0) : String(Number(value.toFixed(2)))

export const formatSiteValue = (r: SiteRating, value: number) =>
  r.scale_max >= 100 ? value.toFixed(1) : value.toFixed(2)

export const formatVotes = (n: number) =>
  n >= 10000
    ? `${Number((n / 10000).toFixed(1))} 万`
    : n.toLocaleString('en-US')

const formatShare = (share: number) =>
  share < 0.1 ? '0.1%' : `${Number(share.toFixed(1))}%`

export const formatTopShare = (percentile: number) =>
  percentile >= 50
    ? `前 ${formatShare(100 - percentile)}`
    : `后 ${formatShare(percentile)}`

export const percentileTone = (percentile: number) => {
  if (percentile >= 99) {
    return 'text-warning-600'
  }
  if (percentile >= 95) {
    return 'text-success-600'
  }
  if (percentile >= 80) {
    return 'text-primary-600'
  }
  if (percentile >= 30) {
    return 'text-default-700'
  }
  return 'text-danger-600'
}
