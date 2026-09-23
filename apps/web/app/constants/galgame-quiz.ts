import type { KunUIColor } from '@kungal/ui-core'
import type {
  QuizCategory,
  QuizSpoilerLevel,
  QuizType
} from '#shared/utils/api/schemas'

export const KUN_QUIZ_SPOILER_CONST = ['none', 'portion', 'serious'] as const

export const KUN_QUIZ_SPOILER_MAP: Record<QuizSpoilerLevel, string> = {
  none: '无剧透',
  portion: '部分剧透',
  serious: '严重剧透'
}

export const KUN_QUIZ_SPOILER_COLOR_MAP: Record<QuizSpoilerLevel, KunUIColor> = {
  none: 'success',
  portion: 'warning',
  serious: 'danger'
}

export const KUN_QUIZ_TYPE_CONST = ['single', 'multiple', 'judge'] as const

export const KUN_QUIZ_TYPE_MAP: Record<QuizType, string> = {
  single: '单选',
  multiple: '多选',
  judge: '判断'
}

export const KUN_QUIZ_TYPE_ICON_MAP: Record<QuizType, string> = {
  single: 'lucide:circle-dot',
  multiple: 'lucide:list-checks',
  judge: 'lucide:scale'
}

export const KUN_QUIZ_TYPE_COLOR_MAP: Record<QuizType, KunUIColor> = {
  single: 'primary',
  multiple: 'secondary',
  judge: 'success'
}

export const KUN_QUIZ_TYPE_DESCRIPTION_MAP: Record<QuizType, string> = {
  single: '给出多个选项, 只有一个正确答案',
  multiple: '给出多个选项, 有一个或多个正确答案',
  judge: '判断一句话是否正确'
}

export const KUN_QUIZ_CATEGORY_CONST = [
  'plot',
  'character',
  'system',
  'music',
  'voice',
  'company',
  'trivia',
  'other'
] as const

export const KUN_QUIZ_CATEGORY_MAP: Record<QuizCategory, string> = {
  plot: '剧情',
  character: '角色',
  system: '系统',
  music: '音乐',
  voice: '声优',
  company: '会社',
  trivia: '常识',
  other: '其他'
}

export const KUN_QUIZ_SORT_FIELD_CONST = [
  'bumped_at',
  'created',
  'view',
  'view_1d',
  'view_7d',
  'view_30d',
  'difficulty',
  'answer_count'
] as const

export const KUN_QUIZ_PROMPT_MAX = 200
export const KUN_QUIZ_CHOICE_MAX = 200
export const KUN_QUIZ_CHOICE_LIMIT = 20
export const KUN_QUIZ_WORK_LIMIT = 20

export const kunQuizDifficultyLabel = (d: number): string => {
  if (d <= 2) return '入门'
  if (d <= 4) return '简单'
  if (d <= 6) return '普通'
  if (d <= 8) return '困难'
  return '地狱'
}

export const kunQuizDifficultyColor = (d: number): KunUIColor => {
  if (d <= 2) return 'success'
  if (d <= 4) return 'primary'
  if (d <= 6) return 'secondary'
  if (d <= 8) return 'warning'
  return 'danger'
}
