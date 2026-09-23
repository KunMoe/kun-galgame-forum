import { z } from 'zod'
import {
  KUN_TOOLSET_TYPE_CONST,
  KUN_TOOLSET_LANGUAGE_CONST,
  KUN_TOOLSET_PLATFORM_CONST,
  KUN_TOOLSET_VERSION_CONST
} from '~/constants/toolset'
import { ResourceSizePattern } from '#shared/utils/pattern'

export const createToolsetSchema = z.object({
  title: z.string().min(1).max(500),
  content_markdown: z.string().max(2000).default(''),
  interface_language: z.enum(KUN_TOOLSET_LANGUAGE_CONST, {
    message: '非法的语言'
  }),
  platform: z.enum(KUN_TOOLSET_PLATFORM_CONST, { message: '非法的平台' }),
  toolset_type: z.enum(KUN_TOOLSET_TYPE_CONST, { message: '非法的工具类型' }),
  release_channel: z.enum(KUN_TOOLSET_VERSION_CONST, {
    message: '非法的版本类型'
  }),
  homepage_urls: z.array(z.url().max(500)).max(10).default([]),
  aliases: z.array(z.string().min(1).max(500)).max(17).default([])
})

export const updateToolsetSchema = createToolsetSchema.merge(
  z.object({
    toolset_id: z.string().min(1)
  })
)

export const createToolsetResourceSchema = z
  .object({
    toolset_resource_type: z.enum(['file', 'link']),
    artifact_id: z.string().optional(),
    link_url: z.string().max(1007).optional(),
    size_label: z.string().max(107).optional(),
    extraction_code: z.string().max(1007).optional(),
    archive_password: z.string().max(1007).optional(),
    note: z.string().max(1007).nullable().optional()
  })
  .superRefine((val, ctx) => {
    if (val.toolset_resource_type === 'file') {
      if (!val.artifact_id) {
        ctx.addIssue({
          code: 'custom',
          path: ['artifact_id'],
          message: '请先上传文件'
        })
      }
      return
    }
    if (!val.link_url) {
      ctx.addIssue({
        code: 'custom',
        path: ['link_url'],
        message: '请填写资源链接'
      })
    }
    if (!val.size_label || !ResourceSizePattern.test(val.size_label)) {
      ctx.addIssue({
        code: 'custom',
        path: ['size_label'],
        message: '大小格式不正确, 需要包含 KB, MB, GB'
      })
    }
  })

export const updateToolsetResourceSchema = z
  .object({
    link_url: z.string().min(1).max(1007).optional(),
    size_label: z.string().max(107).optional(),
    extraction_code: z.string().max(1007).optional(),
    archive_password: z.string().max(1007).optional(),
    note: z.string().max(1007).nullable().optional()
  })
  .superRefine((val, ctx) => {
    if (
      val.size_label !== undefined &&
      !ResourceSizePattern.test(val.size_label)
    ) {
      ctx.addIssue({
        code: 'custom',
        path: ['size_label'],
        message: '大小格式不正确, 需要包含 KB, MB, GB'
      })
    }
  })

export const initToolsetUploadSchema = z.object({
  filename: z
    .string()
    .min(1)
    .max(1007)
    .regex(/\.(7z|zip|rar)$/i, {
      message: '文件名必须以 .7z, .zip 或 .rar 结尾'
    }),
  file_size: z.coerce.number<number>().int().min(1).max(2147483648),
  content_type: z.string().max(100).optional()
})

export const completeToolsetUploadSchema = z.object({
  state: z.literal('completed'),
  parts: z
    .array(z.object({ part_number: z.number().int().min(1), etag: z.string() }))
    .optional()
})
