import type {
  RatedWorkRankingQuery,
  RatedWorkRankingSort,
  RatingSite
} from '#shared/utils/api/schemas'
import { PLATFORM_OPTIONS } from '#shared/utils/galgameResourceVocab'
import {
  RATED_RANKING_LIMIT,
  RATED_RANKING_PRESETS,
  ratedRankingPresets
} from '~/constants/ratedRanking'

type Platform = NonNullable<RatedWorkRankingQuery['platforms']>[number]

const PLATFORMS = new Set<string>(PLATFORM_OPTIONS.map((o) => o.value))

const RATED_RANKING_FILTER_SCHEMA = {
  page: queryPage(),
  sort: queryEnum(
    RATED_RANKING_PRESETS.map((p) => p.sort),
    'lower_desc' as RatedWorkRankingSort
  ),
  votes: queryString(),
  playtime: queryString(),
  circulation: queryEnum(['', 'commercial', 'doujin'] as const, ''),
  platforms: queryList((item) =>
    PLATFORMS.has(item) ? (item as Platform) : undefined
  ),
  minSites: queryString(),
  scope: queryEnum(['', 'all'] as const, ''),
  tags: queryIds(),
  company: queryInt(0, 0),
  releasedFrom: queryString(),
  releasedTo: queryString()
}

export const parseCountRange = (raw: string) => {
  const [lo = '', hi = ''] = raw.split('-')
  const min = Number(lo)
  const max = Number(hi)
  return {
    min: Number.isInteger(min) && min > 0 ? min : 0,
    max: Number.isInteger(max) && max > 0 ? max : 0
  }
}

export const ratedRankingPreset = (
  site: RatingSite | null,
  sort: RatedWorkRankingSort
) => ratedRankingPresets(site).find((p) => p.sort === sort)

export const useRatedRankingFilters = () =>
  useQueryState(RATED_RANKING_FILTER_SCHEMA, { pageKey: 'page' })

export const useRatedRankingQuery = (site: () => RatingSite | null) => {
  const filters = useRatedRankingFilters()
  const { allowsNsfw } = useContentStance()
  const { excludeQuery } = useTagBlocks()
  const { showKUNGalgameAllOriginalLanguages } = storeToRefs(
    usePersistSettingsStore()
  )

  const sort = computed(() => {
    const wanted = filters.state.value.sort
    return ratedRankingPreset(site(), wanted) ? wanted : 'lower_desc'
  })

  const query = computed<RatedWorkRankingQuery>(() => {
    const f = filters.state.value
    const by = site()
    const votes = parseCountRange(f.votes)
    const playtime = parseCountRange(f.playtime)
    const minSites = by ? 0 : parseCountRange(f.minSites).min
    return {
      page: f.page,
      limit: RATED_RANKING_LIMIT,
      sort: sort.value,
      ...(by ? { site: by } : {}),
      ...(votes.min ? { min_vote_count: votes.min } : {}),
      ...(votes.max ? { max_vote_count: votes.max } : {}),
      ...(playtime.min ? { min_playtime_minutes: playtime.min } : {}),
      ...(playtime.max ? { max_playtime_minutes: playtime.max } : {}),
      ...(minSites ? { min_site_count: minSites } : {}),
      ...(f.circulation ? { circulation: f.circulation } : {}),
      ...(f.platforms.length ? { platforms: f.platforms } : {}),
      ...(f.releasedFrom ? { released_from: f.releasedFrom } : {}),
      ...(f.releasedTo ? { released_to: f.releasedTo } : {}),
      ...(f.scope === 'all' ? { include_all_games: true } : {}),
      ...(f.tags.length ? { tag_ids: f.tags.map(String) } : {}),
      ...(f.company ? { company_id: String(f.company) } : {}),
      include_nsfw: allowsNsfw.value,
      include_all_original_languages: showKUNGalgameAllOriginalLanguages.value,
      ...excludeQuery.value
    }
  })

  return { ...filters, sort, query }
}
