import { z } from 'zod'
import {
  KUN_GALGAME_RATING_RECOMMEND_CONST,
  KUN_GALGAME_RATING_SPOILER_CONST,
  KUN_GALGAME_RATING_GAME_TYPE_CONST,
  KUN_GALGAME_RATING_SUMMARY_MAX_RUNES
} from '~/constants/galgame-rating'
import { markdownToText } from '#shared/utils/markdownToText'

const aspect = z.number().int().min(1).max(10).nullable()

export const galgameRatingFormSchema = z
  .object({
    recommend: z.enum(KUN_GALGAME_RATING_RECOMMEND_CONST),
    overall: z.number().int().min(1).max(10),
    game_types: z
      .array(z.enum(KUN_GALGAME_RATING_GAME_TYPE_CONST))
      .min(1, { message: '请至少选择一个' }),
    spoiler_level: z.enum(KUN_GALGAME_RATING_SPOILER_CONST),
    short_summary: z
      .string()
      .refine((s) => [...s].length <= KUN_GALGAME_RATING_SUMMARY_MAX_RUNES, {
        message: `短评最多 ${KUN_GALGAME_RATING_SUMMARY_MAX_RUNES} 个字符 (含 Markdown 标记)`
      }),
    aspect_scores: z.object({
      art: aspect,
      story: aspect,
      music: aspect,
      character: aspect,
      route: aspect,
      system: aspect,
      voice: aspect,
      replay_value: aspect
    })
  })
  .superRefine((data, ctx) => {
    if (
      (data.overall === 1 || data.overall === 10) &&
      markdownToText(data.short_summary).trim().length < 100
    ) {
      ctx.addIssue({
        code: 'custom',
        message: '当总分为 1 或 10 分时需不少于 100 字',
        path: ['short_summary']
      })
    }
  })
