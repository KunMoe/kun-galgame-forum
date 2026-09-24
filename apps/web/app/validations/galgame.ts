import { z } from 'zod'
import {
  KUN_RESOURCE_TYPE_CONST,
  KUN_RESOURCE_LANGUAGE_CONST,
  KUN_RESOURCE_PLATFORM_CONST
} from '~/constants/galgame'
import { PROVIDER_KEY_OPTIONS } from '~/constants/galgameResource'

const SORT_ORDER_CONST = ['asc', 'desc'] as const

const ProviderEnum = z.enum(PROVIDER_KEY_OPTIONS)

const providerQueryArray = z.preprocess((v) => {
  if (Array.isArray(v)) {
    return v
  }
  if (typeof v === 'string') {
    if (!v) return []
    return v.split(',')
  }
  return []
}, z.array(ProviderEnum).default([]))

export const getGalgameSchema = z.object({
  page: z.coerce.number<number>().min(1).max(9999999),
  limit: z.coerce.number<number>().min(1).max(24),
  type: z.enum([...KUN_RESOURCE_TYPE_CONST, 'all']),
  language: z.enum([...KUN_RESOURCE_LANGUAGE_CONST, 'all']),
  platform: z.enum([...KUN_RESOURCE_PLATFORM_CONST, 'all']),
  sort_field: z.enum([
    'time',
    'created',
    'view',
    'view_1d',
    'view_7d',
    'view_30d',
    'release_date',
    'rating'
  ]),
  sort_order: z.enum(SORT_ORDER_CONST),
  include_providers: providerQueryArray,
  exclude_only_providers: providerQueryArray
})

const SUBMISSION_TITLE_MAX = 500
const SUBMISSION_INTRO_MAX = 50000
const SUBMISSION_TITLES_TOTAL = 100

const submissionTitle = z
  .string()
  .max(SUBMISSION_TITLE_MAX, {
    message: `游戏名称最多 ${SUBMISSION_TITLE_MAX} 字`
  })
  .default('')

const submissionIntro = z
  .string()
  .max(SUBMISSION_INTRO_MAX, {
    message: `游戏介绍最多 ${SUBMISSION_INTRO_MAX} 字`
  })
  .default('')

export const submitGalgameSchema = z
  .object({
    name_en_us: submissionTitle,
    name_ja_jp: submissionTitle,
    name_zh_cn: submissionTitle,
    name_zh_tw: submissionTitle,
    intro_en_us: submissionIntro,
    intro_ja_jp: submissionIntro,
    intro_zh_cn: submissionIntro,
    intro_zh_tw: submissionIntro,
    content_limit: z.enum(['sfw', 'nsfw'], { error: '请选择 SFW 或 NSFW' }),
    release_date: z
      .string()
      .refine((v) => v === '' || /^\d{4}-\d{2}-\d{2}$/.test(v), {
        message: '发售日期格式应为 YYYY-MM-DD 或留空'
      })
      .default(''),
    aliases: z
      .array(
        z.string().max(SUBMISSION_TITLE_MAX, {
          message: `每个 Galgame 别名最多 ${SUBMISSION_TITLE_MAX} 个字符`
        })
      )
      .default([])
  })
  .superRefine((data, ctx) => {
    const names = [
      data.name_en_us,
      data.name_ja_jp,
      data.name_zh_cn,
      data.name_zh_tw
    ].filter((n) => n.trim())
    if (!names.length) {
      ctx.addIssue({
        code: 'custom',
        message: '至少需要填写一个语言版本的游戏名称',
        path: ['name_zh_cn']
      })
    }
    const aliasCount = data.aliases.filter((a) => a.trim()).length
    if (names.length + aliasCount > SUBMISSION_TITLES_TOTAL) {
      ctx.addIssue({
        code: 'custom',
        message: `游戏名称与别名合计最多 ${SUBMISSION_TITLES_TOTAL} 个`,
        path: ['aliases']
      })
    }

    const hasAtLeastOneIntro =
      data.intro_en_us.trim() ||
      data.intro_ja_jp.trim() ||
      data.intro_zh_cn.trim() ||
      data.intro_zh_tw.trim()
    if (!hasAtLeastOneIntro) {
      ctx.addIssue({
        code: 'custom',
        message: '至少需要填写一个语言版本的游戏介绍',
        path: ['intro_zh_cn']
      })
    }
  })

export const getGalgameResourceSchema = z.object({
  galgame_id: z.coerce.number<number>().min(1).max(9999999)
})

export const getGalgameResourceDetailSchema = z.object({
  galgame_resource_id: z.coerce.number<number>().min(1).max(9999999)
})
