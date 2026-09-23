import type {
  TopicRankingSort,
  UserRankingSort,
  WorkRankingSort
} from '#shared/utils/api/schemas'

export const RANKING_LIMIT = 50

export const topicRankingPageData = reactive<{ sort: TopicRankingSort }>({
  sort: 'views_desc'
})

export const galgameRankingPageData = reactive<{ sort: WorkRankingSort }>({
  sort: 'views_desc'
})

export const userRankingPageData = reactive<{ sort: UserRankingSort }>({
  sort: 'moemoepoint_desc'
})

export const getRankClasses = (index: number) => {
  if (index === 0) {
    return 'bg-warning-400/20 border-warning-500/50'
  }
  if (index === 1) {
    return 'bg-default-400/20 border-default-500/50'
  }
  if (index === 2) {
    return 'bg-info-400/20 border-info-500/50'
  }
  return 'bg-content1'
}
