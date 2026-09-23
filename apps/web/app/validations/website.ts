import { z } from 'zod'

const DOMAIN_RE =
  /^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]?$/

const hostField = z
  .string()
  .min(1, '网站主域名不能为空')
  .max(253, '网站主域名最多 253 个字符')
  .transform((value) =>
    value
      .trim()
      .toLowerCase()
      .replace(/^https?:\/\//, '')
      .replace(/\/.*$/, '')
      .replace(/\.$/, '')
  )
  .refine((value) => DOMAIN_RE.test(value), {
    message: '无效的网站主域名 (示例: www.kungal.com)'
  })

const siteUrlField = z
  .string()
  .max(100, '网站地址最多 100 个字符')
  .refine((value) => /^https?:\/\/[^\s/]+/.test(value), {
    message: '网站地址需要以 http:// 或 https:// 开头'
  })

const slugField = (what: string) =>
  z
    .string()
    .min(1, `${what}不能为空`)
    .max(30, `${what}最多 30 个字符`)
    .regex(/^[a-z0-9_-]+$/, `${what}是 URL 的一部分, 只能用小写字母/数字/-/_`)

const labelField = (what: string) =>
  z
    .string()
    .max(30, `${what}最多 30 个字符`)
    .refine((v) => v.trim() !== '', `${what}不能为空`)

export const websiteFormSchema = z.object({
  host: hostField,
  title: z
    .string()
    .max(233, '网站名称最多 233 个字符')
    .refine((v) => v.trim() !== '', '网站名称不能为空'),
  description: z
    .string()
    .max(1000, '网站介绍最多 1000 个字符')
    .refine((v) => v.trim().length >= 10, '网站介绍最少 10 个字符'),
  icon_image_hash: z.string().regex(/^([0-9a-f]{64})?$/, '图标无效'),
  website_category_id: z.string().min(1, '请选择分类'),
  website_tag_ids: z.array(z.string()).max(20, '网站最多 20 个标签'),
  is_nsfw: z.boolean(),
  state: z.enum(['normal', 'unreachable', 'closed']),
  language: z.string().regex(/^[a-z]{2,3}(-[a-z0-9]{2,8})*$/, '语言无效'),
  urls: z.array(siteUrlField).max(10, '可用地址最多 10 个'),
  founded: z.string().max(20, '网站创建时间描述最多 20 个字符')
})

export const websiteCategoryFormSchema = z.object({
  slug: slugField('分类名称'),
  label: labelField('分类显示名'),
  description: z.string().max(300, '网站分类描述最多 300 个字符'),
  sort_order: z.coerce.number<number>().int().min(0).max(9999)
})

export const websiteTagFormSchema = z.object({
  slug: slugField('标签名称'),
  label: labelField('标签显示名'),
  description: z.string().max(300, '网站标签描述最多 300 个字符'),
  level: z.coerce
    .number<number>()
    .int('标签等级必须是整数')
    .min(-100, '网站标签等级最小为 -100')
    .max(20, '网站标签等级最大为 20'),
  website_tag_group_id: z.string().nullable()
})

export const websiteTagGroupFormSchema = z.object({
  slug: slugField('分组名称'),
  label: labelField('分组显示名'),
  description: z.string().max(300, '分组描述最多 300 个字符'),
  sort_order: z.coerce.number<number>().int().min(0).max(9999),
  is_multi_select: z.boolean()
})
