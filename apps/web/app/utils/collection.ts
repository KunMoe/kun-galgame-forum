import { fieldMessage, problemMessage } from '#shared/utils/api/message'
import type { ClientProblem } from '#shared/utils/api/problem'
import type { WorkRef } from '#shared/utils/api/schemas'

export const collectionDisplayName = (
  c: { is_default: boolean; title: string },
  ownerName?: string | null
): string => {
  if (c.is_default && !c.title) {
    return `${ownerName || '我'}的收藏夹`
  }
  return c.title
}

export const collectionDisplayDescription = (
  c: { is_default: boolean; description: string },
  ownerName?: string | null
): string => {
  if (c.is_default && !c.description) {
    const who = ownerName || '我'
    return `${who}所有收藏的游戏已经被放置在 ${who}的收藏夹`
  }
  return c.description
}

export const COLLECTION_DIRECTORY_SORTS = [
  { value: 'followers_desc', textValue: '最多关注', icon: 'lucide:users' },
  { value: 'updated_desc', textValue: '最近更新', icon: 'lucide:refresh-cw' },
  { value: 'created_desc', textValue: '最新创建', icon: 'lucide:sparkles' }
] as const

export const COLLECTION_REPORT_CATEGORIES = [
  { value: 'spam', label: '垃圾广告' },
  { value: 'abuse', label: '辱骂 / 人身攻击' },
  { value: 'rating_mislabel', label: '分级错误 (含未标注的 R18 内容)' },
  { value: 'illegal', label: '违法违规' },
  { value: 'copyright', label: '侵犯版权' },
  { value: 'other', label: '其它' }
] as const

export const COLLECTION_NOTE_MAX = 500

// A problem's field error rendered under the form field it points at. A
// CONTENT_REJECTED from the forum's own word list names no field, so the
// caller says which one the text came from.
export const collectionFieldErrors = (
  problem: ClientProblem,
  fallbackPointer?: string
): Record<string, string> => {
  const out: Record<string, string> = {}
  for (const error of problem.errors) {
    if (error.pointer) {
      out[error.pointer] = fieldMessage(error)
    }
  }
  if (
    !Object.keys(out).length &&
    fallbackPointer &&
    problem.code === 'CONTENT_REJECTED'
  ) {
    out[fallbackPointer] = problemMessage(problem)
  }
  return out
}

export const collectionBlurb = (
  c: { description: string; recent_works: WorkRef[] },
  nameOf: (work: WorkRef) => string
): string => {
  if (c.description.trim()) {
    return c.description
  }
  const names = c.recent_works.map((w) => nameOf(w))
  return names.length ? `最近收藏：${names.join('、')}` : ''
}

export const COLLECTION_VISIBILITY_OPTIONS = [
  { value: 'public', label: '公开 · 出现在收藏夹广场和个人主页' },
  { value: 'unlisted', label: '仅链接可见 · 拿到链接的人才能看到' },
  { value: 'private', label: '私密 · 仅自己可见' }
] as const

export const collectionVisibilityMeta = (v?: string) => {
  switch (v) {
    case 'private':
      return { icon: 'lucide:lock', label: '私密' }
    case 'unlisted':
      return { icon: 'lucide:link', label: '仅链接可见' }
    default:
      return { icon: 'lucide:globe', label: '公开' }
  }
}

export const COLLECTION_TAG_MAX = 5
export const COLLECTION_TAG_LENGTH_MAX = 16

// The picked cover, else the four newest works as a mosaic. Hidden-NSFW
// readers get each work's SFW portrait.
export const collectionCoverArts = (
  c: {
    cover_work: WorkRef | null
    recent_works: WorkRef[]
    content_rating: string
  },
  blurNsfw: boolean
): string[] => {
  const art = (w: WorkRef) => {
    const masked = blurNsfw && (w.is_nsfw || c.content_rating === 'r18')
    return (masked ? w.cover_sfw : w.cover)?.url ?? ''
  }
  if (c.cover_work) {
    const url = art(c.cover_work)
    if (url) {
      return [url]
    }
  }
  return c.recent_works.map(art).filter(Boolean).slice(0, 4)
}
