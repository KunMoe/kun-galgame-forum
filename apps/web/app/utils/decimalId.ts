export const compareDecimalIds = (left: string, right: string): number => {
  const a = left.replace(/^0+/, '') || '0'
  const b = right.replace(/^0+/, '') || '0'
  if (a.length !== b.length) {
    return a.length - b.length
  }
  if (a === b) {
    return 0
  }
  return a < b ? -1 : 1
}

export const maxDecimalId = (ids: readonly string[]): string | undefined => {
  let max: string | undefined
  for (const id of ids) {
    if (max === undefined || compareDecimalIds(id, max) > 0) {
      max = id
    }
  }
  return max
}
