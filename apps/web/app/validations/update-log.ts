import { z } from 'zod'
import { KUN_UPDATE_LOG_CHANGE_TYPES } from '~/constants/update'

export const updateLogSchema = z.object({
  change_type: z.enum(KUN_UPDATE_LOG_CHANGE_TYPES),
  release_version: z
    .string()
    .max(20, '更新版本号最多 20 个字符')
    .refine((v) => v.trim() !== '', '请填写版本号'),
  text: z
    .string()
    .max(1000, '更新描述最多 1000 个字符')
    .refine((v) => v.trim() !== '', '请填写更新内容')
})
