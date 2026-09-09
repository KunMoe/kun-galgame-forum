export const KUN_GALGAME_PLAYTIME_SOURCE_CONST = [
  'nextmoe',
  'vndb',
  'erogamescape'
] as const

export type KunGalgamePlaytimeSource =
  (typeof KUN_GALGAME_PLAYTIME_SOURCE_CONST)[number]

export interface KunGalgamePlaytimeMeta {
  short: string
  hint: string
  // Catalog writes vote_count 0 for a source that publishes no per-work count,
  // so a 0 means "unknown", never "nobody reported it". Only a source that
  // really counts can be held to minVotes.
  hasVoteCount: boolean
  minVotes: number
}

// Catalog's own wording for the number: a median, on the source's own terms.
// Upstream applies no vote floor to vndb — a work whose length rests on one
// person's guess still ships — so the floor is drawn here, at the same 3
// reporters infra requires before it will publish our own median.
export const KUN_GALGAME_PLAYTIME_SOURCE_MAP: Record<
  KunGalgamePlaytimeSource,
  KunGalgamePlaytimeMeta
> = {
  nextmoe: {
    short: '本站',
    hint: '本站玩家上报通关时长的中位数',
    hasVoteCount: true,
    minVotes: 3
  },
  vndb: {
    short: 'VNDB',
    hint: 'VNDB 用户时长投票的中位数',
    hasVoteCount: true,
    minVotes: 3
  },
  erogamescape: {
    short: '批评空间',
    hint: '批评空间社区统计的中位数, 该来源不公开投票人数',
    hasVoteCount: false,
    minVotes: 0
  }
}

export const KUN_GALGAME_PLAY_STATE_CONST = [
  'wish',
  'doing',
  'done_one_route',
  'done_main',
  'done_all',
  'on_hold',
  'dropped'
] as const
export type KunGalgamePlayState = (typeof KUN_GALGAME_PLAY_STATE_CONST)[number]

// catalog answers state=done with no completion when another application
// reported a completion it does not carry. The forum never writes this value and
// never offers it in a picker; it exists so that reading such a row renders a
// badge instead of a blank.
export type KunGalgamePlayStateRead = KunGalgamePlayState | 'done'

export const KUN_GALGAME_PLAY_STATE_MAP: Record<
  KunGalgamePlayStateRead,
  string
> = {
  wish: '想玩',
  doing: '游玩中',
  done_one_route: '单线通关',
  done_main: '主线通关',
  done_all: '全线通关',
  done: '已通关',
  on_hold: '搁置中',
  dropped: '已弃坑'
}

export const KUN_GALGAME_PLAY_STATE_OPTIONS = [
  { value: 'wish', label: '想玩', icon: 'lucide:bookmark' },
  { value: 'doing', label: '游玩中', icon: 'lucide:play' },
  { value: 'done_one_route', label: '单线通关', icon: 'lucide:flag' },
  { value: 'done_main', label: '主线通关', icon: 'lucide:flag' },
  { value: 'done_all', label: '全线通关', icon: 'lucide:trophy' },
  { value: 'on_hold', label: '搁置中', icon: 'lucide:pause' },
  { value: 'dropped', label: '已弃坑', icon: 'lucide:x' }
] as const

export const KUN_GALGAME_PLAY_STATE_DONE = [
  'done_one_route',
  'done_main',
  'done_all'
] as const

// Both floors are catalog's, not ours: it refuses anything above the ceiling,
// and its aggregate ignores anything under the floor — which is how a report
// gets withdrawn from the public median without a delete endpoint.
export const KUN_GALGAME_PLAYTIME_HOURS_MAX = 1000
export const KUN_GALGAME_PLAYTIME_MINUTES_FLOOR = 10
