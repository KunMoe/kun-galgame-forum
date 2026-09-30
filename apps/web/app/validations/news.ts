import { z } from 'zod'

export const KUN_NEWS_TITLE_MAX = 200
export const KUN_NEWS_PREVIEW_MAX = 200
export const KUN_NEWS_CONTENT_MAX = 20000
export const KUN_NEWS_SOURCE_URL_MAX = 1024

export const newsSubmissionSchema = z
  .object({
    title: z
      .string()
      .trim()
      .min(1, { message: '请填写情报标题' })
      .max(KUN_NEWS_TITLE_MAX, {
        message: `标题不能超过 ${KUN_NEWS_TITLE_MAX} 个字`
      }),
    preview: z
      .string()
      .trim()
      .min(1, { message: '请填写导语' })
      .max(KUN_NEWS_PREVIEW_MAX, {
        message: `导语不能超过 ${KUN_NEWS_PREVIEW_MAX} 个字`
      }),
    content_markdown: z.string().max(KUN_NEWS_CONTENT_MAX, {
      message: `正文不能超过 ${KUN_NEWS_CONTENT_MAX} 个字`
    }),
    source_url: z
      .string()
      .trim()
      .max(KUN_NEWS_SOURCE_URL_MAX, { message: '原文链接过长' })
      .refine((url) => url === '' || /^https?:\/\/\S+$/i.test(url), {
        message: '原文链接必须以 http:// 或 https:// 开头'
      })
  })
  .refine(
    (data) => data.content_markdown.trim() !== '' || data.source_url !== '',
    { message: '正文和原文链接至少要填写一项', path: ['content_markdown'] }
  )
