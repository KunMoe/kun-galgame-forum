import { z } from 'zod'

const prizeSchema = z.object({
  name: z.string().min(1, '奖项名称不能为空').max(100, '奖项名称最多100个字符'),
  description: z.string().max(500, '奖项描述最多500个字符').default(''),
  image_hashes: z
    .array(z.string().regex(/^[0-9a-f]{64}$/))
    .max(9, '单个奖项最多 9 张图片')
    .default([]),
  nsfw_hashes: z
    .array(z.string().regex(/^[0-9a-f]{64}$/))
    .max(9, '单个奖项最多 9 张图片')
    .default([]),
  delivery: z.enum(['code', 'offline', 'point'], {
    message: '奖品发放方式不正确'
  }),
  point_mode: z.enum(['fixed', 'split', 'random'], {
    message: '萌萌点发放方式不正确'
  }),
  point_amount: z.coerce.number<number>().int().min(0).max(100000),
  slots: z.coerce
    .number<number>()
    .int()
    .min(1, '名额至少为 1')
    .max(500, '单个奖项名额最多 500'),
  codes: z.string().default('')
})

export const lotterySchema = z.object({
  title: z
    .string()
    .min(1, '抽奖标题不能为空')
    .max(100, '抽奖标题最多100个字符'),
  description: z.string().max(1000, '抽奖说明最多1000个字符').default(''),
  entry_mode: z.enum(['signup', 'reply', 'floor'], {
    message: '参与方式不正确'
  }),
  floor_rule: z.string().max(200, '楼层规则最多200个字符').default(''),
  draw_mode: z.enum(['deadline', 'manual', 'threshold'], {
    message: '开奖方式不正确'
  }),
  draw_threshold: z.coerce.number<number>().int().min(0).max(100000),
  deadline: z.iso.date().optional(),
  min_account_age_days: z.coerce.number<number>().int().min(0).max(3650),
  min_moemoepoint: z.coerce.number<number>().int().min(0).max(1000000),
  show_entrants: z.boolean(),
  prizes: z
    .array(prizeSchema)
    .min(1, '至少需要一个奖项')
    .max(10, '最多只能有 10 个奖项')
})
