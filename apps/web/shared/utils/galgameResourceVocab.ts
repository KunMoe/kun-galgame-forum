import type { GalgameResource } from './api/schemas'

export interface VocabOption<T extends string = string> {
  value: T
  label: string
}

export const RESOURCE_TYPE_OPTIONS: VocabOption<GalgameResource['resource_type']>[] = [
  { value: 'game', label: '游戏本体' },
  { value: 'collection', label: '合集' },
  { value: 'patch', label: '补丁' },
  { value: 'crack_fix', label: '免CD / 修正' },
  { value: 'mod', label: 'MOD' },
  { value: 'tool', label: '工具' },
  { value: 'walkthrough', label: '攻略' },
  { value: 'ost', label: '原声带' },
  { value: 'voice', label: '语音' },
  { value: 'cg', label: '鉴赏 CG' },
  { value: 'wallpaper', label: '壁纸' },
  { value: 'artbook', label: '设定资料' },
  { value: 'video', label: '视频相关' },
  { value: 'other', label: '其它' }
]

export const LEGACY_TYPE_LABELS: Record<string, string> = {
  image: '图片相关',
  ai: 'AI 相关',
  others: '其它'
}

// Links shared before the entity pages moved to v1 still carry the legacy
// resource scalars; they keep working as the axis key they meant.
export const LEGACY_RESOURCE_PLATFORM: Record<string, string> = {
  windows: 'win',
  app: 'and',
  linux: 'lin',
  others: 'oth'
}
export const LEGACY_RESOURCE_LANGUAGE: Record<string, string> = {
  others: 'other'
}
export const LEGACY_RESOURCE_TYPE: Record<string, string> = {
  others: 'other',
  image: 'cg'
}

export const axisKey = <T extends string>(
  value: string,
  legacy: Record<string, string>,
  options: { value: string }[]
): T | undefined => {
  if (!value || value === 'all') {
    return undefined
  }
  const key = legacy[value] ?? value
  return options.some((o) => o.value === key) ? (key as T) : undefined
}

export const LANGUAGE_OPTIONS: VocabOption<
  GalgameResource['resource_languages'][number]
>[] = [
  { value: 'zh-cn', label: '简体中文' },
  { value: 'zh-tw', label: '繁体中文' },
  { value: 'ja-jp', label: '日语' },
  { value: 'en-us', label: '英语' },
  { value: 'other', label: '其他语言' }
]

export const PLATFORM_OPTIONS: VocabOption<
  GalgameResource['resource_platforms'][number]
>[] = [
  { value: 'win', label: 'Windows 电脑版' },
  { value: 'and', label: '安卓手机版' },
  { value: 'ios', label: 'iOS' },
  { value: 'mac', label: 'macOS' },
  { value: 'lin', label: 'Linux' },
  { value: 'web', label: '网页版' },
  { value: 'mob', label: '功能机' },
  { value: 'swi', label: 'Nintendo Switch' },
  { value: 'sw2', label: 'Nintendo Switch 2' },
  { value: 'n3d', label: 'Nintendo 3DS' },
  { value: 'nds', label: 'Nintendo DS' },
  { value: 'wii', label: 'Wii' },
  { value: 'wiu', label: 'Wii U' },
  { value: 'gba', label: 'Game Boy Advance' },
  { value: 'gbc', label: 'Game Boy Color' },
  { value: 'nes', label: '红白机 Famicom' },
  { value: 'sfc', label: '超任 Super Famicom' },
  { value: 'ps1', label: 'PlayStation' },
  { value: 'ps2', label: 'PlayStation 2' },
  { value: 'ps3', label: 'PlayStation 3' },
  { value: 'ps4', label: 'PlayStation 4' },
  { value: 'ps5', label: 'PlayStation 5' },
  { value: 'psp', label: 'PlayStation Portable' },
  { value: 'psv', label: 'PlayStation Vita' },
  { value: 'xb1', label: 'Xbox' },
  { value: 'xb3', label: 'Xbox 360' },
  { value: 'xbo', label: 'Xbox One' },
  { value: 'xxs', label: 'Xbox Series X/S' },
  { value: 'sat', label: '世嘉土星 Saturn' },
  { value: 'smd', label: 'Mega Drive' },
  { value: 'scd', label: 'Mega-CD' },
  { value: 'drc', label: 'Dreamcast' },
  { value: 'pce', label: 'PC Engine' },
  { value: 'pcf', label: 'PC-FX' },
  { value: 'tdo', label: '3DO' },
  { value: 'p88', label: 'PC-88' },
  { value: 'p98', label: 'PC-98' },
  { value: 'x1s', label: 'Sharp X1' },
  { value: 'x68', label: 'Sharp X68000' },
  { value: 'fm7', label: 'FM-7' },
  { value: 'fm8', label: 'FM-8' },
  { value: 'fmt', label: 'FM Towns' },
  { value: 'msx', label: 'MSX' },
  { value: 'dos', label: 'DOS' },
  { value: 'dvd', label: 'DVD 播放机' },
  { value: 'bdp', label: '蓝光播放机' },
  { value: 'vnd', label: 'VNDS' },
  { value: 'oth', label: '其它平台' }
]

export const RUNTIME_OPTIONS: VocabOption<
  GalgameResource['resource_runtimes'][number]
>[] = [
  { value: 'native-win', label: 'Windows 原生' },
  { value: 'native-and', label: '安卓直装' },
  { value: 'native-ios', label: 'iOS 原生' },
  { value: 'winlator', label: 'Winlator' },
  { value: 'gamehub', label: '盖世游戏 GameHub' },
  { value: 'kirikiroid2', label: 'Kirikiroid2 / XP3Player' },
  { value: 'krkrsdl2', label: 'krkrsdl2（需逐作打包）' },
  { value: 'onscripter', label: 'ONScripter (ONS)' },
  { value: 'joiplay', label: 'JoiPlay' },
  { value: 'easyrpg', label: 'EasyRPG' },
  { value: 'renpy-android', label: "Ren'Py 安卓版" },
  { value: 'tyranor', label: 'Tyranor' },
  { value: 'tyranor-next', label: 'Tyranor Next' },
  { value: 'other', label: '其它运行环境' }
]

export const VERSION_LABEL_OPTIONS: VocabOption<
  NonNullable<GalgameResource['version_label']>
>[] = [
  { value: 'official_latest', label: '官方最新' },
  { value: 'stable', label: '稳定版' },
  { value: 'mirror', label: '镜像版' },
  { value: 'localized', label: '汉化版' },
  { value: 'unknown', label: '未知版本' }
]

const RUNTIME_RELEVANT = new Set([
  'game',
  'collection',
  'patch',
  'crack_fix',
  'mod',
  'tool'
])

export const hasRuntimeAxis = (resourceType: string): boolean =>
  RUNTIME_RELEVANT.has(resourceType)

const labelsOf = (options: VocabOption[]): Record<string, string> =>
  Object.fromEntries(options.map((o) => [o.value, o.label]))

export const RESOURCE_TYPE_LABELS: Record<string, string> = {
  ...labelsOf(RESOURCE_TYPE_OPTIONS),
  ...LEGACY_TYPE_LABELS
}

export const LANGUAGE_LABELS: Record<string, string> = {
  ...labelsOf(LANGUAGE_OPTIONS),
  others: '其他语言'
}

export const PLATFORM_LABELS: Record<string, string> = {
  ...labelsOf(PLATFORM_OPTIONS),
  windows: 'Windows',
  linux: 'Linux',
  emulator: '模拟器',
  app: '应用直装',
  others: '其它'
}

export const RUNTIME_LABELS = labelsOf(RUNTIME_OPTIONS)

export const VERSION_LABELS = labelsOf(VERSION_LABEL_OPTIONS)

export const resourceTypeLabel = (key: string) =>
  RESOURCE_TYPE_LABELS[key] || key

export const resourceLanguageLabel = (key: string) =>
  LANGUAGE_LABELS[key] || key

export const resourcePlatformLabel = (key: string) =>
  PLATFORM_LABELS[key] || key

export const resourceRuntimeLabel = (key: string) =>
  RUNTIME_LABELS[key] || key

export const resourceVersionLabel = (key: string) =>
  VERSION_LABELS[key] || key
