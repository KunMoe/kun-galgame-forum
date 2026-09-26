import type { ActivityType } from '#shared/utils/api/schemas'

export const KUN_ACTIVITY_TYPE_TYPE: Record<ActivityType, string> = {
  galgame_creation: 'Galgame',
  galgame_rating_creation: 'Galgame 评分',
  galgame_rating_comment_creation: 'Galgame 评分评论',
  topic_creation: '新话题',
  topic_upvote: '话题被推',
  best_answer_set: '最佳答案',
  topic_reply_creation: '话题回复',
  topic_comment_creation: '话题评论',
  galgame_website_creation: 'Galgame 网站',
  galgame_resource_creation: 'Galgame 资源',
  galgame_quiz_creation: 'Galgame 题目',
  galgame_edit: 'Galgame 编辑',
  galgame_pr_creation: '提出更新请求',
  galgame_comment_creation: 'Galgame 评论',
  galgame_website_comment_creation: 'Galgame 网站评论',
  todo_creation: '待办',
  update_log_creation: '更新日志',
  toolset_creation: 'Galgame 工具',
  toolset_resource_creation: '工具资源',
  toolset_comment_creation: '工具评论',
  galgame_resource_comment_creation: 'Galgame 资源评论',
  galgame_quiz_comment_creation: 'Galgame 题目讨论'
}

export const KUN_ACTIVITY_GROUPS: { label: string; types: ActivityType[] }[] = [
  {
    label: 'Galgame',
    types: [
      'galgame_creation',
      'galgame_edit',
      'galgame_pr_creation',
      'galgame_resource_creation',
      'galgame_resource_comment_creation',
      'galgame_quiz_creation',
      'galgame_quiz_comment_creation',
      'galgame_rating_creation',
      'galgame_rating_comment_creation',
      'galgame_comment_creation',
      'galgame_website_creation',
      'galgame_website_comment_creation'
    ]
  },
  {
    label: '社区',
    types: [
      'topic_creation',
      'topic_reply_creation',
      'topic_comment_creation',
      'topic_upvote',
      'best_answer_set'
    ]
  },
  {
    label: '工具集',
    types: [
      'toolset_creation',
      'toolset_resource_creation',
      'toolset_comment_creation'
    ]
  },
  {
    label: '站务',
    types: ['todo_creation', 'update_log_creation']
  }
]

export const KUN_ACTIVITY_ICON_MAP: Record<ActivityType, string> = {
  galgame_creation: 'lucide:gamepad-2',
  galgame_rating_creation: 'lucide:star',
  galgame_rating_comment_creation: 'lucide:message-square-text',
  galgame_comment_creation: 'lucide:message-square',
  galgame_website_creation: 'lucide:globe',
  galgame_website_comment_creation: 'lucide:message-square-text',
  galgame_resource_creation: 'lucide:box',
  galgame_quiz_creation: 'lucide:brain',
  galgame_resource_comment_creation: 'lucide:message-square-text',
  galgame_quiz_comment_creation: 'lucide:message-square-text',
  galgame_edit: 'lucide:file-pen-line',
  galgame_pr_creation: 'lucide:git-pull-request',
  toolset_creation: 'lucide:wrench',
  toolset_resource_creation: 'lucide:package-plus',
  toolset_comment_creation: 'lucide:wrench',
  topic_creation: 'icon-park-outline:topic',
  topic_upvote: 'lucide:trending-up',
  topic_reply_creation: 'carbon:reply',
  topic_comment_creation: 'lucide:message-circle-more',
  todo_creation: 'lucide:list-checks',
  update_log_creation: 'lucide:file-clock',
  best_answer_set: 'lucide:bookmark-check'
}

export interface KunFeedKind {
  value: string
  label: string
  icon: string
}

export const KUN_FEED_KIND_GROUPS: { label: string; kinds: KunFeedKind[] }[] = [
  {
    label: '话题',
    kinds: [
      { value: 'TOPIC_NORMAL', label: '话题', icon: 'icon-park-outline:topic' },
      {
        value: 'TOPIC_RESOURCE_HELP',
        label: '资源/求助话题',
        icon: 'lucide:life-buoy'
      },
      {
        value: 'TOPIC_REPLY_CREATION',
        label: '话题回复',
        icon: 'carbon:reply'
      },
      {
        value: 'TOPIC_COMMENT_CREATION',
        label: '话题评论',
        icon: 'lucide:message-circle-more'
      },
      { value: 'TOPIC_UPVOTE', label: '推话题', icon: 'lucide:trending-up' },
      {
        value: 'MESSAGE_SOLUTION',
        label: '最佳答案',
        icon: 'lucide:bookmark-check'
      }
    ]
  },
  {
    label: 'Galgame',
    kinds: [
      { value: 'GALGAME_CREATION', label: '新游戏', icon: 'lucide:gamepad-2' },
      {
        value: 'GALGAME_EDIT',
        label: '游戏编辑',
        icon: 'lucide:file-pen-line'
      },
      {
        value: 'GALGAME_PR_CREATION',
        label: '更新请求',
        icon: 'lucide:git-pull-request'
      },
      {
        value: 'GALGAME_COMMENT_CREATION',
        label: '游戏评论',
        icon: 'lucide:message-square'
      },
      {
        value: 'GALGAME_RATING_CREATION',
        label: '游戏评分',
        icon: 'lucide:star'
      },
      {
        value: 'GALGAME_RATING_COMMENT_CREATION',
        label: '评分评论',
        icon: 'lucide:message-square-text'
      },
      {
        value: 'GALGAME_RESOURCE_CREATION',
        label: '游戏资源',
        icon: 'lucide:box'
      },
      {
        value: 'GALGAME_QUIZ_CREATION',
        label: '游戏题目',
        icon: 'lucide:brain'
      },
      {
        value: 'GALGAME_WEBSITE_CREATION',
        label: '网站',
        icon: 'lucide:globe'
      },
      {
        value: 'GALGAME_WEBSITE_COMMENT_CREATION',
        label: '网站评论',
        icon: 'lucide:message-square-text'
      }
    ]
  },
  {
    label: '工具',
    kinds: [
      { value: 'TOOLSET_CREATION', label: '工具', icon: 'lucide:wrench' },
      {
        value: 'TOOLSET_RESOURCE_CREATION',
        label: '工具资源',
        icon: 'lucide:package-plus'
      },
      {
        value: 'TOOLSET_COMMENT_CREATION',
        label: '工具评论',
        icon: 'lucide:wrench'
      }
    ]
  },
  {
    label: '站务',
    kinds: [
      { value: 'TODO_CREATION', label: '待办', icon: 'lucide:list-checks' },
      {
        value: 'UPDATE_LOG_CREATION',
        label: '更新日志',
        icon: 'lucide:file-clock'
      }
    ]
  }
]

// A tab reads either this site's own activity feed or an outside face. An
// activity tab is defined by its `kinds`; a news tab has none, because its
// content comes from the partner index rather than from feed_activity.
export type KunFeedSource = 'activity' | 'news' | 'following'

export interface KunFeedTab {
  id: string
  name: string
  icon: string
  kinds: string[]
  source?: KunFeedSource
}

export const isNewsFeedTab = (tab?: KunFeedTab): boolean =>
  tab?.source === 'news'

export const isFollowingFeedTab = (tab?: KunFeedTab): boolean =>
  tab?.source === 'following'

const KUN_ALL_TAB_KINDS = [
  'TOPIC_NORMAL',
  'TOPIC_REPLY_CREATION',
  'TOPIC_COMMENT_CREATION',
  'TOPIC_UPVOTE',
  'MESSAGE_SOLUTION',
  'GALGAME_QUIZ_CREATION',
  'GALGAME_QUIZ_COMMENT_CREATION',
  'GALGAME_RESOURCE_COMMENT_CREATION',
  'GALGAME_RATING_CREATION',
  'GALGAME_RATING_COMMENT_CREATION',
  'GALGAME_WEBSITE_CREATION',
  'GALGAME_WEBSITE_COMMENT_CREATION',
  'TOOLSET_CREATION',
  'TOOLSET_RESOURCE_CREATION',
  'TOOLSET_COMMENT_CREATION',
  'TODO_CREATION',
  'UPDATE_LOG_CREATION'
]

const KUN_FOLLOWING_TAB_KINDS = [
  ...KUN_FEED_KIND_GROUPS.flatMap((group) => group.kinds.map((k) => k.value)),
  'GALGAME_RESOURCE_COMMENT_CREATION',
  'GALGAME_QUIZ_COMMENT_CREATION'
]

export const KUN_FEED_TABS_VERSION = 8

export const KUN_DEFAULT_FEED_TABS: KunFeedTab[] = [
  {
    id: 'topic',
    name: '话题',
    icon: 'icon-park-outline:topic',
    kinds: ['TOPIC_NORMAL']
  },
  {
    id: 'following',
    name: '关注',
    icon: 'lucide:user-round-check',
    kinds: [...KUN_FOLLOWING_TAB_KINDS],
    source: 'following'
  },
  {
    id: 'galgame',
    name: 'Galgame',
    icon: 'lucide:gamepad-2',
    kinds: [
      'GALGAME_CREATION',
      'GALGAME_EDIT',
      'GALGAME_PR_CREATION',
      'GALGAME_COMMENT_CREATION',
      'GALGAME_QUIZ_CREATION',
      'GALGAME_QUIZ_COMMENT_CREATION',
      'GALGAME_RATING_CREATION',
      'GALGAME_RATING_COMMENT_CREATION',
      'GALGAME_WEBSITE_CREATION',
      'GALGAME_WEBSITE_COMMENT_CREATION',
      'TOOLSET_CREATION',
      'TOOLSET_RESOURCE_CREATION',
      'TOOLSET_COMMENT_CREATION'
    ]
  },
  {
    id: 'all',
    name: '全站动态',
    icon: 'lucide:layers',
    kinds: [...KUN_ALL_TAB_KINDS]
  },
  {
    id: 'news',
    name: 'Gal 情报',
    icon: 'lucide:newspaper',
    kinds: [],
    source: 'news'
  },
  {
    id: 'resource',
    name: 'Gal 资源',
    icon: 'lucide:box',
    kinds: ['GALGAME_RESOURCE_CREATION', 'GALGAME_RESOURCE_COMMENT_CREATION']
  },
  {
    id: 'resource-help-topic',
    name: '资源和求助',
    icon: 'lucide:life-buoy',
    kinds: ['TOPIC_RESOURCE_HELP']
  },
  {
    id: 'others',
    name: '其他',
    icon: 'lucide:layout-grid',
    kinds: ['TODO_CREATION', 'UPDATE_LOG_CREATION']
  }
]

export const upgradeFeedTabs = (
  tabs: KunFeedTab[],
  version: number
): KunFeedTab[] => {
  if (version < 7) {
    return structuredClone(KUN_DEFAULT_FEED_TABS)
  }
  const have = new Set(tabs.map((t) => t.id))
  const out = [...tabs]
  KUN_DEFAULT_FEED_TABS.forEach((tab, i) => {
    if (!have.has(tab.id)) {
      out.splice(Math.min(i, out.length), 0, structuredClone(tab))
    }
  })
  return out
}
