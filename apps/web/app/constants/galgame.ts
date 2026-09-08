export type KunGalgameResourceTypeOptions =
  | 'all'
  | 'game'
  | 'patch'
  | 'collection'
  | 'voice'
  | 'image'
  | 'ai'
  | 'video'
  | 'others'

export type KunGalgameResourceLanguageOptions =
  | 'all'
  | 'ja-jp'
  | 'en-us'
  | 'zh-cn'
  | 'zh-tw'
  | 'others'

export type KunGalgameResourcePlatformOptions =
  | 'all'
  | 'windows'
  | 'mac'
  | 'linux'
  | 'emulator'
  | 'app'
  | 'others'

export const KUN_GALGAME_RESOURCE_TYPE_MAP: Record<string, string> = {
  name: '资源链接的类型',
  all: '全部类型',
  game: '游戏本体',
  patch: '补丁',
  collection: '合集',
  voice: '音声相关',
  image: '图片相关',
  ai: 'AI 相关',
  video: '视频相关',
  others: '其它'
}
export const kunGalgameResourceTypeOptions = [
  { value: 'all', label: '全部类型' },
  { value: 'game', label: '游戏本体' },
  { value: 'patch', label: '补丁' },
  { value: 'collection', label: '合集' },
  { value: 'voice', label: '音声相关' },
  { value: 'image', label: '图片相关' },
  { value: 'ai', label: 'AI 相关' },
  { value: 'video', label: '视频相关' },
  { value: 'others', label: '其它' }
] as const
export const KUN_RESOURCE_TYPE_CONST = [
  'game',
  'patch',
  'collection',
  'voice',
  'image',
  'ai',
  'video',
  'others'
] as const

export const KUN_GALGAME_RESOURCE_LANGUAGE_MAP: Record<string, string> = {
  all: '全部语言',
  'ja-jp': '日语',
  'en-us': '英语',
  'zh-cn': '简体中文',
  'zh-tw': '繁体中文',
  others: '其它'
}
export const kunGalgameResourceLanguageOptions = [
  { value: 'all', label: '全部语言' },
  { value: 'ja-jp', label: '日语' },
  { value: 'en-us', label: '英语' },
  { value: 'zh-cn', label: '简体中文' },
  { value: 'zh-tw', label: '繁体中文' },
  { value: 'others', label: '其它' }
] as const
export const KUN_RESOURCE_LANGUAGE_CONST = [
  'ja-jp',
  'en-us',
  'zh-cn',
  'zh-tw',
  'others'
] as const

export const KUN_GALGAME_ORIGINAL_LANGUAGE_MAP: Record<string, string> = {
  'ja-jp': '日语',
  'en-us': '英语',
  'zh-cn': '简体中文',
  'zh-tw': '繁体中文',
  'ko-kr': '韩语',
  ru: '俄语',
  es: '西班牙语',
  uk: '乌克兰语',
  fr: '法语',
  de: '德语',
  it: '意大利语',
  'pt-br': '葡萄牙语 (巴西)',
  'pt-pt': '葡萄牙语',
  pl: '波兰语',
  id: '印尼语',
  th: '泰语',
  vi: '越南语',
  tr: '土耳其语',
  cs: '捷克语',
  ar: '阿拉伯语',
  nl: '荷兰语',
  sv: '瑞典语',
  others: '其它'
}

export const kunGalgameOriginalLanguageOptions = Object.entries(
  KUN_GALGAME_ORIGINAL_LANGUAGE_MAP
).map(([value, label]) => ({ value, label }))

export const getGalgameOriginalLanguageName = (langCode: string): string =>
  KUN_GALGAME_ORIGINAL_LANGUAGE_MAP[langCode?.toLowerCase()] || langCode

export const KUN_GALGAME_RESOURCE_PLATFORM_MAP: Record<string, string> = {
  name: '资源链接的平台',
  all: '全部平台',
  windows: 'Windows',
  mac: 'macOS',
  linux: 'Linux',
  emulator: '模拟器',
  app: '应用直装',
  others: '其它'
}
export const kunGalgameResourcePlatformOptions = [
  { value: 'all', label: '全部平台' },
  { value: 'windows', label: 'Windows' },
  { value: 'mac', label: 'macOS' },
  { value: 'linux', label: 'Linux' },
  { value: 'emulator', label: '模拟器' },
  { value: 'app', label: '应用直装' },
  { value: 'others', label: '其它' }
] as const
export const KUN_RESOURCE_PLATFORM_CONST = [
  'windows',
  'mac',
  'linux',
  'emulator',
  'app',
  'others'
] as const

export const KUN_GALGAME_RESOURCE_SORT_FIELD_MAP: Record<string, string> = {
  views: '总浏览数',
  time: '更新顺序',
  created: '创建顺序',
  view_1d: '日浏览数',
  view_7d: '周浏览数',
  view_30d: '月浏览数',
  release_date: '发售日期',
  rating: '评分'
}

// The library reads the catalog, which sorts by its own popularity, its own
// updated_at and the release date — none of the forum-side counters.
export const KUN_GALGAME_LIBRARY_SORT_FIELD_MAP: Record<string, string> = {
  popularity: '热度',
  release_date: '发售日期',
  time: '资料更新'
}

export const KUN_GALGAME_INTRO_LANGUAGE_MAP: Record<string, string> = {
  'zh-Hans': '简体中文',
  'zh-Hant': '繁体中文',
  ja: '日语',
  en: '英语'
}

export const getGalgameIntroLanguageName = (lang: string): string =>
  KUN_GALGAME_INTRO_LANGUAGE_MAP[lang] ?? lang

export const KUN_GALGAME_AGE_LIMIT_MAP: Record<string, string> = {
  all: '本游戏不含有成人内容',
  r18: '本游戏可能含有成人内容'
}

export const KUN_GALGAME_CONTENT_LIMIT_MAP: Record<string, string> = {
  sfw: '页面内容不含有 R18 内容',
  nsfw: '页面内容可能含有 R18 内容'
}

// KunChip's base is `whitespace-nowrap` with the flex default `min-width: auto`,
// so a chip holding text a user typed can neither wrap nor shrink. 「资源体积」 is
// free text (`max=107`) and uploaders write whole titles into it: on /galgame/60
// 「命运石之门5部合集【PC+安卓直装 民汉和官中 附通关存档】32.8gb」 measured 398px
// inside a 260px column at a 390px viewport, and was the *only* reason the whole
// document scrolled sideways (scrollWidth 456 = that chip's right edge). Chip
// merges className through tailwind-merge, so `whitespace-normal` replaces the
// base rather than fighting it.
export const KUN_USER_TEXT_CHIP_CLASS =
  'min-w-0 max-w-full whitespace-normal break-words'
