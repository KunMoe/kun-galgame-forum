import { differenceInSeconds, differenceInHours, format } from 'date-fns'

const isMeaningfulDate = (d: Date): boolean =>
  !isNaN(d.getTime()) && d.getFullYear() > 1

export const formatTimeDifference = (pastTime: number | Date | string) => {
  const past = new Date(pastTime)
  if (!isMeaningfulDate(past)) {
    return ''
  }
  const now = new Date()
  const diffInSeconds = differenceInSeconds(now, past)

  if (diffInSeconds < 10) {
    return '数秒前'
  }

  if (diffInSeconds < 60) {
    return `${diffInSeconds} 秒前`
  } else if (diffInSeconds < 3600) {
    const minutes = Math.floor(diffInSeconds / 60)
    return `${minutes} 分钟前`
  } else if (diffInSeconds < 86400) {
    const hours = Math.floor(diffInSeconds / 3600)
    return `${hours} 小时前`
  } else if (diffInSeconds < 2592000) {
    const days = Math.floor(diffInSeconds / 86400)
    return `${days} 天前`
  } else if (diffInSeconds < 31536000) {
    const months = Math.floor(diffInSeconds / 2592000)
    return `${months} 个月前`
  } else {
    const years = Math.floor(diffInSeconds / 31536000)
    return `${years} 年前`
  }
}

export const hourDiff = (upvoteTime: number | Date | string, hours: number) => {
  if (upvoteTime === 0 || upvoteTime === undefined) {
    return false
  }

  const currentTime = new Date()
  const time = new Date(upvoteTime)

  return differenceInHours(currentTime, time) <= hours
}

export const formatDurationMinutes = (minutes?: number | null): string => {
  if (!Number.isFinite(minutes) || (minutes as number) <= 0) {
    return ''
  }
  const total = Math.round(minutes as number)
  const hours = Math.floor(total / 60)
  const rest = total % 60

  if (!hours) {
    return `${rest} 分钟`
  }
  return rest ? `${hours} 小时 ${rest} 分钟` : `${hours} 小时`
}

// For a narrow stat cell: one number and its unit, never "N 小时 M 分钟".
export const compactDurationMinutes = (
  minutes?: number | null
): { value: string; unit: '分钟' | '小时' } | null => {
  if (!Number.isFinite(minutes) || (minutes as number) <= 0) {
    return null
  }
  const total = Math.round(minutes as number)
  if (total < 60) {
    return { value: String(total), unit: '分钟' }
  }
  const hours = total / 60
  return {
    value:
      hours < 100
        ? String(Math.round(hours * 10) / 10)
        : String(Math.round(hours)),
    unit: '小时'
  }
}

export const formatDate = (
  time: Date | string | number,
  config?: { isShowYear?: boolean; isPrecise?: boolean }
): string => {
  let formatString = 'MM-dd'

  if (config?.isShowYear) {
    formatString = 'yyyy-MM-dd'
  }

  if (config?.isPrecise) {
    formatString = `${formatString} - HH:mm`
  }

  const d = new Date(time)
  if (!isMeaningfulDate(d)) {
    return ''
  }
  return format(d, formatString)
}

export const toYMD = (raw?: string | null): string => {
  if (!raw) return ''
  const trimmed = String(raw).trim()
  if (!trimmed) return ''

  if (/^\d{4}-\d{2}-\d{2}$/.test(trimmed)) {
    return trimmed
  }

  const isoPrefix = trimmed.match(/^(\d{4}-\d{2}-\d{2})/)
  if (isoPrefix) {
    return isoPrefix[1]!
  }

  return trimmed
}

export const getReleaseDateText = (
  releaseDate?: string | null,
  releaseDateTBA?: boolean
): string => {
  const d = toYMD(releaseDate)
  if (!d) return releaseDateTBA ? '未定 (TBA)' : '未公布'
  return releaseDateTBA ? `预计 ${d}` : d
}
