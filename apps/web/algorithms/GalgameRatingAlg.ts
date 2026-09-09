import type {
  KUN_GALGAME_RATING_RECOMMEND_CONST,
  KUN_GALGAME_DIMENSIONS
} from '~/constants/galgame-rating'
import type { KunGalgamePlayState } from '~/constants/galgame-playtime'

interface DimsInput {
  art: number
  story: number
  music: number
  character: number
  route: number
  system: number
  voice: number
  replay_value: number
}

type KunGalgameRatingDim = (typeof KUN_GALGAME_DIMENSIONS)[number]
type KunGalgameRatingRecommend =
  (typeof KUN_GALGAME_RATING_RECOMMEND_CONST)[number]

const DEFAULT_DIM_WEIGHTS: Record<KunGalgameRatingDim, number> = {
  story: 0.25,
  character: 0.2,
  art: 0.15,
  route: 0.1,
  music: 0.08,
  voice: 0.06,
  system: 0.06,
  replay_value: 0.1
}

const DEFAULT_OVERALL_WEIGHT = 0.4

const RECOMMEND_INFLUENCE_POINTS = 0.8

const RECOMMEND_SCORE_MAP: Record<string, number> = {
  strong_no: -1,
  no: -0.5,
  neutral: 0,
  yes: 0.5,
  strong_yes: 1
}

const PLAY_STATE_ADJUST: Record<KunGalgamePlayState, number> = {
  wish: -1.5,
  doing: -0.8,
  done_one_route: -0.2,
  done_main: 0,
  done_all: 0.4,
  on_hold: -0.9,
  dropped: -1.0
}

export const calcGalgameRating = (
  dims: DimsInput,
  overall: number,
  play_status: KunGalgamePlayState,
  recommend: KunGalgameRatingRecommend
): number => {
  const clamp = (v: number, lo: number, hi: number) =>
    Math.max(lo, Math.min(hi, v))
  const round1 = (v: number) => Math.round(v * 10) / 10

  const safeDims: Record<KunGalgameRatingDim, number> = {
    art: clamp(dims.art ?? 0, 0, 10),
    story: clamp(dims.story ?? 0, 0, 10),
    music: clamp(dims.music ?? 0, 0, 10),
    character: clamp(dims.character ?? 0, 0, 10),
    route: clamp(dims.route ?? 0, 0, 10),
    system: clamp(dims.system ?? 0, 0, 10),
    voice: clamp(dims.voice ?? 0, 0, 10),
    replay_value: clamp(dims.replay_value ?? 0, 0, 10)
  }
  const safeOverall = clamp(overall ?? 0, 0, 10)

  const DIM_KEYS = Object.keys(DEFAULT_DIM_WEIGHTS) as KunGalgameRatingDim[]
  const weightSum =
    DIM_KEYS.reduce((s, k) => s + (DEFAULT_DIM_WEIGHTS[k] ?? 0), 0) || 1
  const dimsWeighted = DIM_KEYS.reduce(
    (s, k) => s + safeDims[k] * ((DEFAULT_DIM_WEIGHTS[k] ?? 0) / weightSum),
    0
  )

  const overallWeight = clamp(DEFAULT_OVERALL_WEIGHT ?? 0.4, 0, 1)
  const dimsWeight = 1 - overallWeight
  const baseScore = safeOverall * overallWeight + dimsWeighted * dimsWeight

  const recVal = RECOMMEND_SCORE_MAP[recommend] ?? 0
  const recAdjust = recVal * RECOMMEND_INFLUENCE_POINTS

  const statusAdjust = PLAY_STATE_ADJUST[play_status] ?? 0

  let finalScore = baseScore + recAdjust + statusAdjust
  finalScore = clamp(finalScore, 0, 10)

  return round1(finalScore)
}
