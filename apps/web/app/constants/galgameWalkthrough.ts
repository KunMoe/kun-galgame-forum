export const KUN_GALGAME_WALKTHROUGH_SOURCE: Record<string, string> = {
  seiya: '誠也の部屋',
  sagaoz_foolmaker: '愚者の館',
  nextmoe: 'NextMoe'
}

export const KUN_GALGAME_WALKTHROUGH_RELEASE_KIND: Record<string, string> = {
  default: '通常版',
  digital: '下载版',
  physical: '实体版',
  trial: '体验版',
  patch: '补丁'
}

export const KUN_GALGAME_WALKTHROUGH_ENDING_TYPE: Record<string, string> = {
  heroine: '个人线',
  harem: '后宫',
  true: '真结局',
  normal: '普通结局',
  bad: '坏结局',
  after: '后日谈',
  other: '其他'
}

export const KUN_GALGAME_WALKTHROUGH_TEXT_MODES = [
  { value: 'both', textValue: '双语' },
  { value: 'ja', textValue: '原文' },
  { value: 'zh', textValue: '中文' }
] as const

export type KunWalkthroughTextMode =
  (typeof KUN_GALGAME_WALKTHROUGH_TEXT_MODES)[number]['value']

export const KUN_GALGAME_WALKTHROUGH_TONES = [
  {
    chip: 'bg-primary-100 text-primary-700 border-primary-300',
    dot: 'bg-primary-500',
    stroke: 'stroke-primary-400',
    fill: 'fill-primary-400',
    border: 'border-primary-400'
  },
  {
    chip: 'bg-secondary-100 text-secondary-700 border-secondary-300',
    dot: 'bg-secondary-500',
    stroke: 'stroke-secondary-400',
    fill: 'fill-secondary-400',
    border: 'border-secondary-400'
  },
  {
    chip: 'bg-success-100 text-success-700 border-success-300',
    dot: 'bg-success-500',
    stroke: 'stroke-success-500',
    fill: 'fill-success-500',
    border: 'border-success-500'
  },
  {
    chip: 'bg-warning-100 text-warning-700 border-warning-300',
    dot: 'bg-warning-500',
    stroke: 'stroke-warning-500',
    fill: 'fill-warning-500',
    border: 'border-warning-500'
  },
  {
    chip: 'bg-danger-100 text-danger-700 border-danger-300',
    dot: 'bg-danger-500',
    stroke: 'stroke-danger-400',
    fill: 'fill-danger-400',
    border: 'border-danger-400'
  },
  {
    chip: 'bg-info-100 text-info-700 border-info-300',
    dot: 'bg-info-500',
    stroke: 'stroke-info-500',
    fill: 'fill-info-500',
    border: 'border-info-500'
  }
] as const

export const KUN_GALGAME_WALKTHROUGH_NEUTRAL_TONE = {
  chip: 'bg-default-100 text-default-700 border-default-300',
  dot: 'bg-default-400',
  stroke: 'stroke-default-300',
  fill: 'fill-default-300',
  border: 'border-default-300'
} as const
