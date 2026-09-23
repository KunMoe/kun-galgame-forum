import {
  KUN_QUIZ_CATEGORY_CONST,
  KUN_QUIZ_CATEGORY_MAP,
  KUN_QUIZ_TYPE_CONST,
  KUN_QUIZ_TYPE_MAP,
  kunQuizDifficultyLabel
} from '~/constants/galgame-quiz'

export const quizCategoryOptions = [
  { value: 'all', label: '全部分类' },
  ...KUN_QUIZ_CATEGORY_CONST.map((c) => ({
    value: c,
    label: KUN_QUIZ_CATEGORY_MAP[c]
  }))
]

export const quizTypeOptions = [
  { value: 'all', label: '全部题型' },
  ...KUN_QUIZ_TYPE_CONST.map((t) => ({
    value: t,
    label: KUN_QUIZ_TYPE_MAP[t]
  }))
]

export const quizDifficultyOptions = [
  { value: 0, label: '全部难度' },
  ...Array.from({ length: 10 }, (_, i) => i + 1).map((n) => ({
    value: n,
    label: `${n} · ${kunQuizDifficultyLabel(n)}`
  }))
]

export const quizSortFieldOptions = [
  { value: 'bumped_at', label: '最近更新' },
  { value: 'created', label: '创建时间' },
  { value: 'view_1d', label: '日浏览数' },
  { value: 'view_7d', label: '周浏览数' },
  { value: 'view_30d', label: '月浏览数' },
  { value: 'view', label: '总浏览数' },
  { value: 'difficulty', label: '难度' },
  { value: 'answer_count', label: '作答数' }
]
