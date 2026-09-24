import type { DocCategory } from '#shared/utils/api/schemas'

export const KUN_DOC_CATEGORIES: DocCategory[] = [
  'galgame',
  'notice',
  'kun',
  'other'
]

export const KUN_DOC_CATEGORY_MAP: Record<DocCategory, string> = {
  galgame: 'Galgame',
  notice: '网站公告',
  kun: '关于鲲',
  other: '其它'
}
