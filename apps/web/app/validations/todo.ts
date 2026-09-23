import { z } from 'zod'
import { KUN_TODO_PROJECTS } from '~/constants/update'

export const todoSchema = z.object({
  project: z.enum(KUN_TODO_PROJECTS),
  text: z
    .string()
    .max(1000, '待办最多 1000 个字符')
    .refine((v) => v.trim() !== '', '请填写待办内容')
})
