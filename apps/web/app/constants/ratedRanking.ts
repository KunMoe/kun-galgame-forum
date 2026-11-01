import type {
  RatedWorkRankingSort,
  RatingSite
} from '#shared/utils/api/schemas'

export interface RatingSiteMeta {
  label: string
  short: string
  scale: string
  description: string
}

export interface RatedRankingPageMeta {
  title: string
  description: string
}

export const RATED_RANKING_CONSENSUS_META: RatedRankingPageMeta = {
  title: 'Galgame 综合评分排行',
  description:
    '综合 VNDB、Bangumi、批评空间、DLsite、HowLongToBeat 与站内评分六个站点, 按跨站共识分为 Galgame 排名, 每部作品附各站原分'
}

export const RATING_SITES: RatingSite[] = [
  'vndb',
  'bangumi',
  'erogamescape',
  'dlsite',
  'howlongtobeat',
  'nextmoe'
]

export const RATING_SITE_META: Record<RatingSite, RatingSiteMeta> = {
  vndb: {
    label: 'VNDB',
    short: 'VNDB',
    scale: '满分 10',
    description: 'VNDB 用户评分的加权平均'
  },
  bangumi: {
    label: 'Bangumi',
    short: 'Bangumi',
    scale: '满分 10',
    description: 'Bangumi 用户评分的平均值'
  },
  erogamescape: {
    label: '批评空间',
    short: '批评空间',
    scale: '满分 100',
    description: '批评空间 (ErogameScape) 用户评分的中位数'
  },
  dlsite: {
    label: 'DLsite',
    short: 'DLsite',
    scale: '满分 5',
    description: 'DLsite 购买者的平均星级'
  },
  howlongtobeat: {
    label: 'HowLongToBeat',
    short: 'HLTB',
    scale: '满分 100',
    description: 'HowLongToBeat 玩家的评价分'
  },
  nextmoe: {
    label: '站内评分',
    short: '站内',
    scale: '满分 10',
    description: '鲲 Galgame 与 NextMoe 各站用户共用的评分'
  }
}

export const ratedRankingPageMeta = (
  site: RatingSite | null
): RatedRankingPageMeta => {
  if (!site) {
    return RATED_RANKING_CONSENSUS_META
  }
  const meta = RATING_SITE_META[site]
  return {
    title:
      site === 'nextmoe'
        ? '鲲 Galgame 站内评分排行'
        : `${meta.label} Galgame 评分排行`,
    description: `按${meta.description} (${meta.scale}) 为 Galgame 排名, 同时列出其它站点的原分与综合评分`
  }
}

export const isRatingSite = (value: unknown): value is RatingSite =>
  typeof value === 'string' && RATING_SITES.includes(value as RatingSite)

export const ratingSiteLabel = (site: string) =>
  isRatingSite(site) ? RATING_SITE_META[site].short : site

export type RatedRankingScope = 'consensus' | 'site' | 'both'

export interface RatedRankingPreset {
  sort: RatedWorkRankingSort
  label: string
  icon: string
  scope: RatedRankingScope
  hint: string
  votes?: string
}

export const RATED_RANKING_PRESETS: RatedRankingPreset[] = [
  {
    sort: 'lower_desc',
    label: '综合榜',
    icon: 'lucide:trophy',
    scope: 'both',
    hint: '确定地好'
  },
  {
    sort: 'native_desc',
    label: '原分榜',
    icon: 'lucide:hash',
    scope: 'site',
    hint: '该站原分',
    votes: '50-'
  },
  {
    sort: 'mean_desc',
    label: '冷门佳作',
    icon: 'lucide:gem',
    scope: 'both',
    hint: '5 - 50 票',
    votes: '5-50'
  },
  {
    sort: 'divergence_desc',
    label: '争议之作',
    icon: 'lucide:split',
    scope: 'consensus',
    hint: '各站分歧大'
  },
  {
    sort: 'spread_desc',
    label: '两极分化',
    icon: 'lucide:split',
    scope: 'site',
    hint: '站内分歧大',
    votes: '100-'
  },
  {
    sort: 'votes_desc',
    label: '人气榜',
    icon: 'lucide:flame',
    scope: 'both',
    hint: '票数最多'
  },
  {
    sort: 'upper_asc',
    label: '地雷榜',
    icon: 'lucide:bomb',
    scope: 'both',
    hint: '确定地差',
    votes: '50-'
  }
]

export const ratedRankingPresets = (site: RatingSite | null) =>
  RATED_RANKING_PRESETS.filter(
    (p) => p.scope === 'both' || p.scope === (site ? 'site' : 'consensus')
  )

export const RATED_RANKING_VOTE_OPTIONS: FilterOption[] = [
  { value: '', label: '不限票数' },
  { value: '10-', label: '至少 10 票' },
  { value: '50-', label: '至少 50 票' },
  { value: '100-', label: '至少 100 票' },
  { value: '500-', label: '至少 500 票' },
  { value: '1000-', label: '至少 1000 票' },
  { value: '5-50', label: '5 - 50 票', hint: '冷门' }
]

export const RATED_RANKING_PLAYTIME_OPTIONS: FilterOption[] = [
  { value: '', label: '不限时长' },
  { value: '0-600', label: '10 小时以内', hint: '短篇' },
  { value: '600-1800', label: '10 - 30 小时', hint: '中篇' },
  { value: '1800-3000', label: '30 - 50 小时', hint: '长篇' },
  { value: '3000-', label: '50 小时以上', hint: '超长篇' }
]

export const RATED_RANKING_CIRCULATION_OPTIONS: FilterOption[] = [
  { value: '', label: '商业与同人' },
  { value: 'commercial', label: '商业作品' },
  { value: 'doujin', label: '同人作品' }
]

export const RATED_RANKING_SITE_COUNT_OPTIONS: FilterOption[] = [
  { value: '', label: '不限站点数' },
  { value: '2', label: '至少 2 个站点评过' },
  { value: '3', label: '至少 3 个站点评过' },
  { value: '4', label: '至少 4 个站点评过' }
]

export const RATED_RANKING_SCOPE_OPTIONS: FilterOption[] = [
  { value: '', label: '仅视觉小说' },
  { value: 'all', label: '全部游戏', hint: '含主机与 PC 游戏' }
]

export const RATED_RANKING_LIMIT = 50
