import { z } from 'zod'

// restricted ("visible to named users") went with the move to catalog folders.
// Production held zero restricted collections and zero named viewers on the day
// it was removed; catalog's unlisted (anyone with the link) is not the same.
export const KUN_COLLECTION_VISIBILITY_CONST = [
  'public',
  'unlisted',
  'private'
] as const

export const createCollectionSchema = z.object({
  title: z
    .string()
    .trim()
    .min(1, { message: '收藏夹名称不能为空' })
    .max(100, { message: '收藏夹名称不能超过 100 个字符' }),
  description: z
    .string()
    .max(500, { message: '收藏夹描述不能超过 500 个字符' })
    .default(''),
  visibility: z.enum(KUN_COLLECTION_VISIBILITY_CONST),
  tags: z
    .array(
      z
        .string()
        .trim()
        .min(1)
        .max(16, { message: '每个标签不能超过 16 个字符' })
    )
    .max(5, { message: '最多 5 个标签' })
    .default([])
})
