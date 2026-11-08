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

// catalog keeps a finish as state=done plus an optional completion; the flat
// value joins the two. A bare done is a finish with the completion unstated.
export type KunGalgamePlayStateRead = KunGalgamePlayState | 'done'

export const KUN_GALGAME_PLAY_STATE_MAP: Record<
  KunGalgamePlayStateRead,
  string
> = {
  wish: '想玩',
  doing: '在玩',
  done_one_route: '单线通关',
  done_main: '主线通关',
  done_all: '全线通关',
  done: '玩过',
  on_hold: '搁置',
  dropped: '弃坑'
}

export const KUN_GALGAME_PLAY_STATE_BASE = [
  { value: 'wish', label: '想玩', icon: 'lucide:bookmark' },
  { value: 'doing', label: '在玩', icon: 'lucide:play' },
  { value: 'done', label: '玩过', icon: 'lucide:flag' },
  { value: 'on_hold', label: '搁置', icon: 'lucide:pause' },
  { value: 'dropped', label: '弃坑', icon: 'lucide:x' }
] as const
export type KunGalgamePlayStateBase =
  (typeof KUN_GALGAME_PLAY_STATE_BASE)[number]['value']

export const KUN_GALGAME_PLAY_COMPLETION_OPTIONS = [
  { value: '', label: '不填' },
  { value: 'done_one_route', label: '单线通关' },
  { value: 'done_main', label: '主线通关' },
  { value: 'done_all', label: '全线通关' }
] as const
export type KunGalgamePlayCompletion =
  (typeof KUN_GALGAME_PLAY_COMPLETION_OPTIONS)[number]['value']

export const KUN_GALGAME_PLAY_STATE_DONE = [
  'done_one_route',
  'done_main',
  'done_all'
] as const

export const kunPlayStateIsDone = (value: string | null | undefined) =>
  value === 'done' ||
  (KUN_GALGAME_PLAY_STATE_DONE as readonly string[]).includes(value ?? '')

export const kunPlayStateBase = (
  value: string | null | undefined
): KunGalgamePlayStateBase | null => {
  if (!value) return null
  if (kunPlayStateIsDone(value)) return 'done'
  return KUN_GALGAME_PLAY_STATE_BASE.some((s) => s.value === value)
    ? (value as KunGalgamePlayStateBase)
    : null
}

export const kunPlayStateColor = (value: string) => {
  if (kunPlayStateIsDone(value)) {
    return 'success'
  }
  if (value === 'dropped') return 'danger'
  return 'default'
}

export const KUN_GALGAME_LIBRARY_TABS = [
  { value: 'all', label: '全部', icon: 'lucide:library' },
  { value: 'wish', label: '想玩', icon: 'lucide:bookmark' },
  { value: 'doing', label: '在玩', icon: 'lucide:play' },
  { value: 'done', label: '玩过', icon: 'lucide:flag' },
  { value: 'on_hold', label: '搁置', icon: 'lucide:pause' },
  { value: 'dropped', label: '弃坑', icon: 'lucide:x' }
] as const

export type KunGalgameLibraryTab =
  (typeof KUN_GALGAME_LIBRARY_TABS)[number]['value']

// Both floors are catalog's, not ours: it refuses anything above the ceiling,
// and its aggregate ignores anything under the floor.
export const KUN_GALGAME_PLAYTIME_HOURS_MAX = 1000
export const KUN_GALGAME_PLAYTIME_MINUTES_FLOOR = 10
