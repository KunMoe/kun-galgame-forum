import type { WebsiteState } from '#shared/utils/api/schemas'

export const KUN_WEBSITE_LANGUAGE_MAP: Record<string, string> = {
  'en-us': '英语',
  'ja-jp': '日语',
  'zh-cn': '简体中文',
  'zh-tw': '繁体中文'
}

export const KUN_WEBSITE_NSFW_OPTIONS = [
  { value: 'sfw', label: '页面默认不含 R18 内容' },
  { value: 'nsfw', label: '含有 R18 内容' }
]

export const KUN_WEBSITE_STATUS_OPTIONS: {
  value: WebsiteState
  label: string
}[] = [
  { value: 'normal', label: '正常' },
  { value: 'unreachable', label: '无法访问' },
  { value: 'closed', label: '已关站' }
]

export const KUN_WEBSITE_STATUS_CHIP: Record<
  WebsiteState,
  { label: string; color: 'warning' | 'danger' } | null
> = {
  normal: null,
  unreachable: { label: '无法访问', color: 'warning' },
  closed: { label: '已关站', color: 'danger' }
}
