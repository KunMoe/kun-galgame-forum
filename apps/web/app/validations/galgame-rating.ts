import { z } from 'zod'
import { KUN_GALGAME_PLAY_STATE_CONST } from '~/constants/galgame-playtime'
import {
  KUN_GALGAME_RATING_RECOMMEND_CONST,
  KUN_GALGAME_RATING_SPOILER_CONST,
  KUN_GALGAME_RATING_GAME_TYPE_CONST,
  KUN_GALGAME_RATING_SORT_FIELD_CONST
} from '~/constants/galgame-rating'

export const createGalgameRatingSchema = z
  .object({
    galgame_id: z.coerce.number<number>().min(1).max(9999999),
    recommend: z.enum(KUN_GALGAME_RATING_RECOMMEND_CONST),
    overall: z.coerce.number<number>().int().min(1).max(10),
    galgame_type: z
      .array(z.enum(KUN_GALGAME_RATING_GAME_TYPE_CONST))
      .min(1, { message: '请至少选择一个' }),
    play_status: z.enum(KUN_GALGAME_PLAY_STATE_CONST),
    short_summary: z
      .string()
      .max(1314, { message: '评价最多 1314 个字符' })
      .default(''),
    spoiler_level: z.enum(KUN_GALGAME_RATING_SPOILER_CONST).default('none'),

    art: z.coerce.number<number>().int().min(0).max(10).default(0),
    story: z.coerce.number<number>().int().min(0).max(10).default(0),
    music: z.coerce.number<number>().int().min(0).max(10).default(0),
    character: z.coerce.number<number>().int().min(0).max(10).default(0),
    route: z.coerce.number<number>().int().min(0).max(10).default(0),
    system: z.coerce.number<number>().int().min(0).max(10).default(0),
    voice: z.coerce.number<number>().int().min(0).max(10).default(0),
    replay_value: z.coerce.number<number>().int().min(0).max(10).default(0)
  })
  .superRefine((data, ctx) => {
    if (
      (data.overall === 1 || data.overall === 10) &&
      (data.short_summary ?? '').trim().length < 100
    ) {
      ctx.addIssue({
        code: 'custom',
        message: '当总分为 1 或 10 分时需不少于 100 字',
        path: ['short_summary']
      })
    }
  })

export const updateGalgameRatingSchema = z
  .object({
    galgame_rating_id: z.coerce.number<number>().min(1).max(9999999),
    recommend: z.enum(KUN_GALGAME_RATING_RECOMMEND_CONST),
    overall: z.coerce.number<number>().int().min(1).max(10),
    galgame_type: z
      .array(z.enum(KUN_GALGAME_RATING_GAME_TYPE_CONST))
      .min(1, { message: '请至少选择一个' }),
    play_status: z.enum(KUN_GALGAME_PLAY_STATE_CONST),
    short_summary: z.string().max(1314, { message: '评价最多 1314 个字符' }),
    spoiler_level: z.enum(KUN_GALGAME_RATING_SPOILER_CONST),

    art: z.coerce.number<number>().int().min(0).max(10),
    story: z.coerce.number<number>().int().min(0).max(10),
    music: z.coerce.number<number>().int().min(0).max(10),
    character: z.coerce.number<number>().int().min(0).max(10),
    route: z.coerce.number<number>().int().min(0).max(10),
    system: z.coerce.number<number>().int().min(0).max(10),
    voice: z.coerce.number<number>().int().min(0).max(10),
    replay_value: z.coerce.number<number>().int().min(0).max(10)
  })
  .superRefine((data, ctx) => {
    if (data.overall !== undefined) {
      const needSummary = data.overall === 1 || data.overall === 10
      if (
        needSummary &&
        (!data.short_summary || data.short_summary.trim().length < 100)
      ) {
        ctx.addIssue({
          code: 'custom',
          message: '当总分为 1 或 10 分时需不少于 100 字',
          path: ['short_summary']
        })
      }
    }
  })
