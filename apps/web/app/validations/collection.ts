import { z } from 'zod'

// restricted ("visible to named users") went with the move to catalog folders,
// which know only private and public. Production held zero restricted
// collections and zero named viewers on the day it was removed.
export const KUN_COLLECTION_VISIBILITY_CONST = ['public', 'private'] as const

export const createCollectionSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, { message: '收藏夹名称不能为空' })
    .max(60, { message: '收藏夹名称不能超过 60 个字符' }),
  description: z
    .string()
    .max(500, { message: '收藏夹描述不能超过 500 个字符' })
    .default(''),
  visibility: z.enum(KUN_COLLECTION_VISIBILITY_CONST)
})
