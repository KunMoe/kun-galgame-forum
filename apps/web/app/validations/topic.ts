import { z } from 'zod'
import { KUN_TOPIC_TITLE_LENGTH_LIMIT } from '~/config/limit'
import {
  KUN_TOPIC_ACCESS_ROLE_CONST,
  KUN_TOPIC_ACCESS_SCOPE_CONST,
  KUN_TOPIC_ACCESS_USER_LIMIT,
  TOPIC_SORT_FIELD_CONST
} from '~/constants/topic'

const SORT_ORDER_CONST = ['asc', 'desc'] as const

const COVER_HASH = /^[0-9a-f]{64}$/
const USER_ID = /^[0-9]+$/

export const getTopicSchema = z.object({
  page: z.coerce.number<number>().min(1).max(9999999),
  limit: z.coerce.number<number>().min(1).max(30),
  sort_field: z.enum(TOPIC_SORT_FIELD_CONST),
  sort_order: z.enum(SORT_ORDER_CONST),
  category: z.enum(['galgame', 'technique', 'others', 'all'])
})

export const createTopicSchema = z
  .object({
    title: z
      .string()
      .min(1, { message: '话题标题最少 1 个字符' })
      .max(KUN_TOPIC_TITLE_LENGTH_LIMIT, {
        message: `话题标题最大长度为 ${KUN_TOPIC_TITLE_LENGTH_LIMIT} 个字符`
      })
      .refine((t) => t.trim().length, { message: '话题标题最少为 1 个字符' }),
    content_markdown: z
      .string()
      .min(1, { message: '话题内容最少 1 个字符' })
      .max(100007, { message: '话题内容最大长度为 100007 个字符' })
      .refine((t) => t.trim().length, { message: '话题内容最少为 1 个字符' }),
    sections: z.array(z.string()).length(1, { message: '请选择话题的分区' }),
    is_nsfw: z.boolean({ message: '未找到话题的 NSFW 设置' }),
    cover_image_hashes: z
      .array(z.string().regex(COVER_HASH, { message: '封面图格式不正确' }))
      .max(9, { message: '封面图最多 9 张' })
      .optional(),
    access_scope: z.enum(KUN_TOPIC_ACCESS_SCOPE_CONST, {
      message: '无效的话题访问范围'
    }),
    access_roles: z
      .array(z.enum(KUN_TOPIC_ACCESS_ROLE_CONST))
      .max(KUN_TOPIC_ACCESS_ROLE_CONST.length, {
        message: `最多指定 ${KUN_TOPIC_ACCESS_ROLE_CONST.length} 个角色`
      })
      .optional(),
    access_user_ids: z
      .array(z.string().regex(USER_ID, { message: '用户 id 格式不正确' }))
      .max(KUN_TOPIC_ACCESS_USER_LIMIT, {
        message: `最多指定 ${KUN_TOPIC_ACCESS_USER_LIMIT} 位用户`
      })
      .optional()
  })
  .superRefine((data, ctx) => {
    if (data.access_scope === 'role' && !data.access_roles?.length) {
      ctx.addIssue({
        code: 'custom',
        message: '请至少选择一个可以看到本话题的角色',
        path: ['access_roles']
      })
    }
    if (data.access_scope === 'users' && !data.access_user_ids?.length) {
      ctx.addIssue({
        code: 'custom',
        message: '请至少指定一位可以看到本话题的用户',
        path: ['access_user_ids']
      })
    }
  })

export const createReplySchema = z.object({
  content_markdown: z
    .string()
    .trim()
    .min(1, { message: '回复内容不能为空' })
    .max(10007, { message: '单条回复的最大长度为 10007 个字符' })
})

export const updateReplySchema = z.object({
  content_markdown: z
    .string()
    .trim()
    .min(1, { message: '回复内容不能为空' })
    .max(10007, { message: '单条回复的最大长度为 10007 个字符' })
})

// K19: the server applies maxLength to the raw value and trims afterwards, so
// counting a trimmed string here would let "1000 characters plus a space"
// through to a 422.
const commentText = z
  .string()
  .max(1007, { message: '单条评论的最大长度为 1007 个字符' })
  .refine((text) => text.trim().length > 0, { message: '评论内容不能为空' })

export const createCommentSchema = z.object({
  text: commentText,
  parent_comment_id: z
    .string()
    .regex(/^[0-9]+$/, { message: '父评论 id 格式不正确' })
    .optional()
})

export const updateCommentSchema = z.object({ text: commentText })
