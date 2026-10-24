import type { SectionColor } from '#shared/utils/api/schemas'

export const SECTION_ICON_CHOICES = [
  'lucide:messages-square',
  'lucide:map',
  'lucide:book-open-text',
  'lucide:newspaper',
  'lucide:search',
  'lucide:archive',
  'lucide:life-buoy',
  'lucide:monitor-play',
  'lucide:code',
  'lucide:sparkles',
  'lucide:coffee',
  'lucide:megaphone',
  'lucide:gamepad-2',
  'lucide:drafting-compass',
  'lucide:circle-ellipsis',
  'lucide:package-search',
  'lucide:music',
  'lucide:film',
  'lucide:pen-line',
  'lucide:wrench',
  'lucide:hash'
] as const

export const SECTION_COLOR_CHOICES: { value: SectionColor; label: string }[] = [
  { value: 'default', label: '默认' },
  { value: 'primary', label: '主色' },
  { value: 'secondary', label: '次色' },
  { value: 'success', label: '绿色' },
  { value: 'warning', label: '黄色' },
  { value: 'danger', label: '红色' }
]
