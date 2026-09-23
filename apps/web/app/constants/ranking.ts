import type { KunTabItem } from '@kungal/ui-vue'
import type {
  TopicRankingSort,
  UserRankingSort,
  WorkRankingSort
} from '#shared/utils/api/schemas'

export interface RankingSortItem<S extends string> {
  icon: string
  label: string
  sort: S
}

export const topicSortItem: RankingSortItem<TopicRankingSort>[] = [
  { icon: 'lucide:eye', label: '浏览数', sort: 'views_desc' },
  { icon: 'carbon:reply', label: '回复数', sort: 'replies_desc' },
  { icon: 'uil:comment-dots', label: '评论数', sort: 'comments_desc' },
  { icon: 'lucide:thumbs-up', label: '点赞数', sort: 'likes_desc' },
  { icon: 'lucide:sparkles', label: '被推数', sort: 'upvotes_desc' },
  { icon: 'lucide:heart', label: '收藏数', sort: 'favorites_desc' }
]

export const galgameSortItem: RankingSortItem<WorkRankingSort>[] = [
  { icon: 'lucide:eye', label: '浏览数', sort: 'views_desc' },
  { icon: 'lucide:thumbs-up', label: '点赞数', sort: 'likes_desc' },
  { icon: 'lucide:heart', label: '收藏数', sort: 'favorites_desc' },
  { icon: 'lucide:box', label: '资源数', sort: 'resources_desc' },
  { icon: 'lucide:star', label: '评分', sort: 'rating_desc' }
]

export const userSortItem: RankingSortItem<UserRankingSort>[] = [
  { icon: 'lucide:lollipop', label: '萌萌点', sort: 'moemoepoint_desc' },
  { icon: 'lucide:square-gantt-chart', label: '话题数', sort: 'topics_desc' },
  { icon: 'carbon:reply', label: '回复数', sort: 'replies_desc' },
  { icon: 'uil:comment-dots', label: '评论数', sort: 'comments_desc' },
  { icon: 'lucide:box', label: 'Galgame 资源', sort: 'resources_desc' }
]

export const rankingPageTabs: KunTabItem[] = [
  {
    textValue: '话题',
    value: 'topic',
    href: '/ranking/topic'
  },
  {
    textValue: 'Galgame',
    value: 'galgame',
    href: '/ranking/galgame'
  },
  {
    textValue: '用户',
    value: 'user',
    href: '/ranking/user'
  }
]

export const rankingPageMetaData: Record<
  string,
  {
    title: string
    description: string
  }
> = {
  topic: {
    title: '话题排行',
    description:
      '查看关于 Galgame 话题的浏览数, 点赞数, 回复数, 评论数排行, 浏览最多人关注的 Galgame 话题'
  },
  galgame: {
    title: 'Galgame 排行',
    description:
      '最强大的 Galgame 排行, 根据本站所有用户的历史数据综合得出, 具有良好的参考价值。适合用于 Galgame, Galgame 资源, Galgame 交流'
  },
  user: {
    title: '用户排行',
    description:
      '用户排行, 查看最强大的 Galgamer, 查看发布 Galgame 资源最多的用户, 查看 Galgamer 相关的动态, Galgame, Galgame 资源与话题'
  }
}
