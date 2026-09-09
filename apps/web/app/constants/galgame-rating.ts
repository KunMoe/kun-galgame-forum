import type { KunUIColor } from '@kungal/ui-core'
import {
  galgameRatingTier,
  type GalgameRatingTierKey,
  type GalgameRatingTierRule
} from '~~/shared/utils/galgameRatingTier'

export const KUN_GALGAME_RATING_RECOMMEND_CONST = [
  'strong_no',
  'no',
  'neutral',
  'yes',
  'strong_yes'
] as const
export const KUN_GALGAME_RATING_RECOMMEND_MAP: Record<string, string> = {
  strong_no: '强烈不推荐',
  no: '不推荐',
  neutral: '中立',
  yes: '推荐',
  strong_yes: '强烈推荐'
}
export const KUN_GALGAME_RATING_RECOMMEND_COLOR_MAP: Record<
  string,
  KunUIColor
> = {
  strong_no: 'danger',
  no: 'default',
  neutral: 'secondary',
  yes: 'success',
  strong_yes: 'warning'
}

export const KUN_GALGAME_RATING_SPOILER_CONST = [
  'none',
  'portion',
  'serious'
] as const
export const KUN_GALGAME_RATING_SPOILER_MAP: Record<string, string> = {
  none: '本评分无剧透',
  portion: '本评分有部分剧透',
  serious: '本评分有严重剧透'
}
export const KUN_GALGAME_RATING_SPOILER_COLOR_MAP: Record<string, KunUIColor> =
  {
    none: 'success',
    portion: 'warning',
    serious: 'danger'
  }

export const KUN_GALGAME_RATING_SPOILER_WARNING =
  '该评分可能含有剧透内容，点进查看'

export const KUN_GALGAME_RATING_GAME_TYPE_CONST = [
  'ba_saku',
  'plot',
  'moe',
  'daily'
] as const
export const KUN_GALGAME_RATING_GAME_TYPE_MAP: Record<string, string> = {
  ba_saku: '拔作',
  plot: '剧情作',
  moe: '萌系',
  daily: '日常系'
}
export const KUN_GALGAME_RATING_GAME_TYPE_DESCRIPTION_MAP: Record<
  string,
  string
> = {
  ba_saku: '以色情为主要游戏内容',
  plot: '作品是否是剧情作由你来定义',
  moe: '画风萌萌的 Galgame 即为萌作',
  daily: '作品风格偏日常恋爱系'
}

export const KUN_GALGAME_RATING_SORT_FIELD_CONST = [
  'time',
  'view',
  'overall'
] as const

export const KUN_GALGAME_EXTERNAL_RATING_CONST = [
  'vndb',
  'bangumi',
  'erogamescape',
  'dlsite'
] as const
export type KunGalgameExternalRatingSource =
  (typeof KUN_GALGAME_EXTERNAL_RATING_CONST)[number]

export const KUN_GALGAME_RATING_TIER_MAP: Record<GalgameRatingTierKey, string> =
  {
    god: '神作',
    masterpiece: '名作',
    good: '良作',
    average: '平作',
    bad: '雷作'
  }
export const KUN_GALGAME_RATING_TIER_COLOR_MAP: Record<
  GalgameRatingTierKey,
  KunUIColor
> = {
  god: 'warning',
  masterpiece: 'success',
  good: 'primary',
  average: 'default',
  bad: 'danger'
}
export const KUN_GALGAME_RATING_TIER_DESCRIPTION_MAP: Record<
  GalgameRatingTierKey,
  string
> = {
  god: '该来源打分最高的一小撮作品',
  masterpiece: '该来源公认的口碑作',
  good: '该来源评价良好, 值得一玩',
  average: '该来源评价平平, 见仁见智',
  bad: '该来源评价偏低, 入坑需谨慎'
}
export const KUN_GALGAME_RATING_TIER_INSUFFICIENT = '评分人数不足'
export const KUN_GALGAME_RATING_TIER_SCOPE_HINT =
  '分级按各来源自己的刻度独立计算, 不跨来源混算'

export interface KunGalgameRatingHistogramAxis {
  // The bucket keys catalog ships for this source, ascending. The histogram
  // arrives sparse, so a key missing from the payload is a real zero.
  keys: number[]
  label: (key: number) => string
  // Why the bars need not add up to the vote_count printed next to them.
  basis?: string
}

export interface KunGalgameExternalRatingMeta {
  label: string
  short: string
  scale: string
  hint: string
  aggregate: string
  max: number
  formatScore: (score: number) => string
  scoreSuffix: string
  tier: GalgameRatingTierRule
  histogram: KunGalgameRatingHistogramAxis
  unit?: string
  link?: (ref: string) => string
}

const kunRatingPointAxis = (max: number): KunGalgameRatingHistogramAxis => ({
  keys: Array.from({ length: max }, (_, index) => index + 1),
  label: String
})

// 批评空间 publishes no histogram of its own; catalog aggregates one from the
// reviews mirror as deciles keyed by the lower bound — 0, 10, … 100, where 100
// is the single top score rather than a range. Folding those keys onto a 1-100
// point axis would leave 90 empty columns and drop the 0 bucket entirely.
const kunRatingDecileAxis: KunGalgameRatingHistogramAxis = {
  keys: [0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100],
  label: (key) => (key === 100 ? '100' : `${key}-${key + 9}`),
  basis: '这张图来自评论表, 和上面的中位数各自独立同步'
}

const kunRatingVndbAxis: KunGalgameRatingHistogramAxis = {
  ...kunRatingPointAxis(10),
  basis: 'VNDB 的投票导出不含私密收藏列表里的票'
}

const kunRatingScoreDecimal = (score: number) =>
  String(Number(score.toFixed(2)))

// 批评空间 quotes a score as the band it falls in ("75点台"), so this truncates
// rather than rounds: 75.8 is still the 75 band.
const kunRatingScoreBand = (score: number) => String(Math.floor(score))

// Catalog ships each source on its own native scale and never normalizes across
// them: vndb/bangumi are a 0-10 mean, dlsite a 0-5 star mean, erogamescape a
// 0-100 median. `max` is the divisor any cross-source comparison has to use.
//
// `tier.cutoffs` are on that same native scale, calibrated per source against
// the whole catalog corpus rather than converted from one shared curve — the
// same work sits at bangumi 7.0 and dlsite 4.5, so one set of numbers cannot
// serve both. dlsite in particular polls self-selected buyers and runs about a
// tier high; that is why its lines are so much tighter than the others'.
export const KUN_GALGAME_EXTERNAL_RATING_MAP: Record<
  KunGalgameExternalRatingSource,
  KunGalgameExternalRatingMeta
> = {
  vndb: {
    label: 'VNDB',
    short: 'VNDB',
    scale: '满分 10',
    // The score is VNDB's c_rating, its weighted headline number — c_average,
    // the plain mean, is what arrives in `stats`. Calling this one 平均值 makes
    // the panel's "上面那个是 X, 和这里的平均分不是同一个数" read as nonsense.
    hint: 'VNDB 用户评分的加权平均',
    aggregate: '加权平均',
    max: 10,
    formatScore: kunRatingScoreDecimal,
    scoreSuffix: '/10',
    tier: { cutoffs: [5.5, 6.5, 7.4, 8.0], minVotes: 10 },
    histogram: kunRatingVndbAxis,
    link: (ref) => `https://vndb.org/${ref}`
  },
  bangumi: {
    label: 'Bangumi',
    short: 'Bangumi',
    scale: '满分 10',
    hint: 'Bangumi 用户评分的平均值',
    aggregate: '平均值',
    max: 10,
    formatScore: kunRatingScoreDecimal,
    scoreSuffix: '/10',
    // Bangumi's own vote labels sit on exactly these integers: 5 不过不失,
    // 6 还行, 7 推荐, 8 力荐.
    tier: { cutoffs: [5.0, 6.0, 7.0, 8.0], minVotes: 10 },
    histogram: kunRatingPointAxis(10),
    link: (ref) => `https://bgm.tv/subject/${ref}`
  },
  erogamescape: {
    label: '批评空间',
    short: '批评空间',
    scale: '满分 100',
    hint: '批评空间用户评分的中位数',
    aggregate: '中位数',
    max: 100,
    formatScore: kunRatingScoreBand,
    scoreSuffix: '点台',
    // 80点 is the line the community itself quotes, and it is kept even though
    // it is stricter here than the other sources' 名作 lines: the famous "80点
    // = 上位34%" figure describes single VOTES, and per-work medians are far
    // more concentrated — only ~8.5% of works clear 80.
    tier: { cutoffs: [60, 70, 80, 85], minVotes: 10 },
    histogram: kunRatingDecileAxis,
    unit: '点',
    link: (ref) =>
      `https://erogamescape.dyndns.org/~ap2/ero/toukei_kaiseki/game.php?game=${ref}`
  },
  dlsite: {
    label: 'DLsite',
    short: 'DLsite',
    scale: '满分 5',
    hint: 'DLsite 购买者的平均星级',
    aggregate: '平均值',
    max: 5,
    formatScore: kunRatingScoreDecimal,
    scoreSuffix: '/5',
    tier: { cutoffs: [3.8, 4.2, 4.5, 4.7], minVotes: 10 },
    histogram: kunRatingPointAxis(5),
    unit: '星'
  }
}

export const KUN_GALGAME_LOCAL_RATING_SOURCE = 'kungal'
export const KUN_GALGAME_LOCAL_RATING_META: KunGalgameExternalRatingMeta = {
  label: '鲲 Galgame 评分',
  short: '鲲 Galgame',
  scale: '满分 10',
  hint: '鲲 Galgame 用户评分的贝叶斯平均',
  aggregate: '贝叶斯平均',
  max: 10,
  formatScore: (score) => score.toFixed(1),
  scoreSuffix: '/10',
  // The bayesian prior already pulls thin samples to the middle, so these lines
  // are much tighter than VNDB's and the gate can afford to be 3 instead of 10.
  tier: { cutoffs: [6.8, 7.3, 7.9, 8.3], minVotes: 3 },
  histogram: kunRatingPointAxis(10)
}

export interface KunGalgameRatingTierBadge {
  key: GalgameRatingTierKey
  label: string
  color: KunUIColor
  description: string
}

export const kunGalgameRatingTierBadge = (
  meta: KunGalgameExternalRatingMeta,
  score: number | null | undefined,
  voteCount: number | null | undefined
): KunGalgameRatingTierBadge | null => {
  const key = galgameRatingTier(meta.tier, score, voteCount)
  if (!key) {
    return null
  }
  return {
    key,
    label: KUN_GALGAME_RATING_TIER_MAP[key],
    color: KUN_GALGAME_RATING_TIER_COLOR_MAP[key],
    description: KUN_GALGAME_RATING_TIER_DESCRIPTION_MAP[key]
  }
}

export const KUN_GALGAME_DIMENSIONS = [
  'art',
  'story',
  'music',
  'character',
  'route',
  'system',
  'voice',
  'replay_value'
] as const
export type KunGalgameDim = (typeof KUN_GALGAME_DIMENSIONS)[number]

export const KUN_GALGAME_DIM_LABELS: Record<KunGalgameDim, string> = {
  art: '画面',
  story: '剧情',
  music: '音乐',
  character: '角色',
  route: '分支',
  system: '系统',
  voice: '配音',
  replay_value: '重玩'
}

export const KUN_GALGAME_DIM_DESCRIPTIONS: Record<KunGalgameDim, string[]> = {
  art: [
    '',
    '无画面, 或者画的简直不是人',
    '作画太简单, 或多处崩坏',
    '画面令人感觉突兀',
    '画面太平凡, 没有一点感觉',
    '画风一般, 中规中矩',
    '部分 CG 还是很不错的',
    '整体画面精美, 细节出色',
    '画面有美学, 部分 CG 堪称经典',
    '顶尖画师, 顶尖作画',
    '最强作画, 这游戏简直就是艺术'
  ],
  story: [
    '',
    '白开水都不如',
    '剧情很迷惑, 不知所以',
    '剧情绝大部分比较无聊',
    '剧情一小半部分比较无聊',
    '剧情一般, 中规中矩',
    '剧情合理, 有一定的情感',
    '小剧情作, 有点沉浸',
    '剧情有大惊喜',
    '剧情非常出色, 百里挑一',
    '简直就是神作, 一辈子都忘不了'
  ],
  music: [
    '',
    '没有音乐, 或者纯噪音',
    'BGM 完全不合适, 听了出戏',
    '音乐存在感极低, 记不住任何旋律',
    'BGM 非常普通, 淡而无味',
    '音乐一般, 基本能配合剧情',
    '偶尔有不错的 BGM, 气氛还行',
    '整体音乐质量很高, 氛围感强',
    '有几首 BGM/ED 特别惊艳',
    '音乐非常出色, 结束之后单独去听',
    '神级 BGM, 直接听哭了'
  ],
  character: [
    '',
    '这游戏塑造的是鸡, 不是人',
    '勉强能算是在写人',
    '很难对这个游戏的角色有印象',
    '部分角色比较无聊, 没什么特色',
    '角色设定一般, 能接受',
    '有几个角色还挺有意思',
    '大部分角色有一定魅力, 可以当老婆',
    '角色个性鲜明, 有必推的人',
    '游戏推完之后好久好久都在想她',
    '我要和她恋爱一辈子, 立马结婚生孩子'
  ],
  route: [
    '',
    '没有分支, 单线游戏',
    '所谓分支只是摆设',
    '路线设计很单调, 重复度高',
    '部分路线很敷衍',
    '路线设计中规中矩',
    '有几个人物路线设计得还不错',
    '各路线剧情都相对完整',
    '分支路线有惊喜, 体验良好',
    '路线设计非常优秀, 选项合理',
    '分支和真结局都是神级, 回味无穷'
  ],
  system: [
    '',
    '系统极其简陋, 甚至有错误',
    '界面交互非常反人类',
    '功能少得可怜, 影响游玩体验',
    '系统功能不够完善',
    '系统一般, 该有的都有',
    '系统设计比较流程',
    '系统设计丝滑, 功能比较实用',
    '系统甚至有贴心的功能设计, 提升体验',
    '系统体验极为丝滑, 各方面都很完善',
    '完美的系统设计, 体验堪比现代大作'
  ],
  voice: [
    '',
    '没有配音, 不是人配的音',
    '配音质量极差, 不完整, 影响体验',
    '配音很敷衍, 听着尴尬',
    '部分角色配音糟糕或无配音',
    '配音中规中矩, 除男主外都有配音',
    '部分角色的声优表现出色',
    '整体配音质量不错, 沉浸感强',
    '声优表现非常亮眼, 完全贴合角色',
    '顶级声优阵容, 听觉享受',
    '神级演绎, 传奇声优阵容'
  ],
  replay_value: [
    '',
    '谁玩第二次谁 **',
    '玩过一遍就腻了',
    '重复体验基本没有必要',
    '部分场景想要重玩',
    '部分人物路线想要重玩',
    '有些场景值得玩很多次',
    '重玩依旧能有乐趣',
    '二刷三刷都有新体验',
    '百玩不厌, 每次都有收获',
    '神作, 一周不看混身难受'
  ]
}
