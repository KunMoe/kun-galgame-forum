import { z } from 'zod'
import {
  KUN_QUIZ_TYPE_CONST,
  KUN_QUIZ_CATEGORY_CONST,
  KUN_QUIZ_SPOILER_CONST,
  KUN_QUIZ_PROMPT_MAX,
  KUN_QUIZ_CHOICE_MAX,
  KUN_QUIZ_CHOICE_LIMIT,
  KUN_QUIZ_WORK_LIMIT
} from '~/constants/galgame-quiz'

export const createGalgameQuizSchema = z.object({
  work_ids: z.array(z.string().min(1)).max(KUN_QUIZ_WORK_LIMIT).default([]),
  is_work_hidden: z.boolean().default(false),
  quiz_category: z.enum(KUN_QUIZ_CATEGORY_CONST),
  quiz_type: z.enum(KUN_QUIZ_TYPE_CONST),
  difficulty: z.coerce.number<number>().int().min(1).max(10),
  spoiler_level: z.enum(KUN_QUIZ_SPOILER_CONST).default('none'),
  prompt_text: z
    .string()
    .min(1, { message: '请填写题干' })
    .max(KUN_QUIZ_PROMPT_MAX, { message: '题干长度不能超过 200 字' }),
  description_markdown: z
    .string()
    .max(20000, { message: '描述长度不能超过 20000 字' })
    .default(''),
  explanation_markdown: z
    .string()
    .max(2000, { message: '解析长度不能超过 2000 字' })
    .default(''),
  choices: z
    .array(z.string().min(1).max(KUN_QUIZ_CHOICE_MAX))
    .max(KUN_QUIZ_CHOICE_LIMIT)
    .optional(),
  correct_choice_indexes: z
    .array(z.number().int().min(0).max(KUN_QUIZ_CHOICE_LIMIT - 1))
    .optional(),
  is_statement_true: z.boolean().nullable().optional()
})

export const rateGalgameQuizQualitySchema = z.object({
  rating: z.coerce.number<number>().int().min(1).max(10)
})
