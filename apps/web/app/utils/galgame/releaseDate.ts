export type WorkReleaseDatePrecision = 'day' | 'month' | 'year' | null

export const workReleaseText = (
  date?: string | null,
  precision?: WorkReleaseDatePrecision
) => {
  if (!date) {
    return '未定 (TBA)'
  }
  if (precision === 'year') {
    return date.slice(0, 4)
  }
  if (precision === 'month') {
    return date.slice(0, 7)
  }
  return getReleaseDateText(date, false)
}
