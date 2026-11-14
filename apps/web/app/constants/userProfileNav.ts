import type { KunTabItem } from '@kungal/ui-vue'
import type { UserProfile } from '#shared/utils/api/schemas'

export type UserNavSection = {
  value: string
  label: string
  href: string
  match: string
  count?: number | null
  hideWhenEmpty?: boolean
}

export type UserNavGroup = {
  label?: string
  sections: UserNavSection[]
}

export type UserNavItem = {
  value: string
  label: string
  icon: string
  groups: UserNavGroup[]
}

export const userProfileNav = (
  user: UserProfile,
  isOwner: boolean
): UserNavItem[] => {
  const base = `/user/${user.id}`
  const c = user.counts
  const section = (
    value: string,
    label: string,
    path: string,
    {
      match = path,
      ...rest
    }: Pick<UserNavSection, 'count' | 'hideWhenEmpty'> & { match?: string } = {}
  ): UserNavSection => ({
    value,
    label,
    href: `${base}/${path}`,
    match: `${base}/${match}`,
    ...rest
  })

  const items: UserNavItem[] = [
    {
      value: 'activity',
      label: '动态',
      icon: 'lucide:activity',
      groups: [{ sections: [section('activity', '动态', 'activity')] }]
    },
    {
      value: 'library',
      label: '游戏库',
      icon: 'lucide:library',
      groups: [
        {
          sections: [
            section('library', '游戏记录', 'library'),
            ...(isOwner ? [section('playtime', '游玩时长', 'playtime')] : [])
          ]
        }
      ]
    },
    {
      value: 'collection',
      label: '收藏',
      icon: 'lucide:heart',
      groups: [
        {
          sections: [
            section('collection-galgame', '游戏收藏夹', 'collection/galgame'),
            section('collection-topic', '话题收藏', 'collection/topic'),
            section('collection-toolset', '工具收藏', 'collection/toolset')
          ]
        }
      ]
    },
    {
      value: 'follow',
      label: '关注',
      icon: 'lucide:users',
      groups: [
        {
          sections: [
            section('following', '关注', 'follow/following', {
              count: c.following_count
            }),
            section('followers', '粉丝', 'follow/followers', {
              count: c.follower_count
            }),
            ...(isOwner
              ? [section('followed-companies', '会社', 'followed-companies')]
              : [])
          ]
        }
      ]
    },
    {
      value: 'publish',
      label: '发布',
      icon: 'lucide:square-pen',
      groups: [
        {
          label: '讨论',
          sections: [
            section('topic', '话题', 'topic/topic', {
              match: 'topic',
              count: c.topic_count
            }),
            section('reply', '回复', 'reply/reply-created', {
              match: 'reply',
              count: c.reply_count
            }),
            section('comment', '评论', 'comment/comment-created', {
              match: 'comment',
              count: c.topic_comment_count
            })
          ]
        },
        {
          label: '作品',
          sections: [
            section('galgame', 'Galgame', 'galgame/galgame-publish', {
              match: 'galgame',
              count: c.published_galgame_count
            }),
            section('rating', '评分', 'rating', {
              count: c.galgame_rating_count,
              hideWhenEmpty: true
            }),
            section('quiz', '题库', 'quiz'),
            section('resource', '资源', 'resource/valid', {
              match: 'resource',
              count: c.galgame_resource_count
            }),
            section('toolset', '工具', 'toolset', {
              count: c.toolset_count,
              hideWhenEmpty: true
            })
          ]
        }
      ]
    },
    {
      value: 'info',
      label: '关于',
      icon: 'lucide:user-round',
      groups: [{ sections: [section('info', '关于', 'info')] }]
    }
  ]

  if (isOwner) {
    items.push({
      value: 'setting',
      label: '设置',
      icon: 'lucide:settings',
      groups: [{ sections: [section('setting', '设置', 'setting')] }]
    })
  }

  if (isOwner) {
    return items
  }
  return items.map((item) => ({
    ...item,
    groups: item.groups
      .map((group) => ({
        ...group,
        sections: group.sections.filter(
          (s) => !(s.hideWhenEmpty && s.count === 0)
        )
      }))
      .filter((group) => group.sections.length)
  }))
}

const matches = (path: string, prefix: string) =>
  path === prefix || path.startsWith(`${prefix}/`)

export const findUserNavSection = (nav: UserNavItem[], path: string) => {
  for (const item of nav) {
    for (const group of item.groups) {
      const hit = group.sections.find((s) => matches(path, s.match))
      if (hit) {
        return { item, section: hit }
      }
    }
  }
  return null
}

export const userNavTabs = (nav: UserNavItem[]): KunTabItem[] =>
  nav.map((item) => ({
    value: item.value,
    textValue: item.label,
    icon: item.icon,
    href: item.groups[0]?.sections[0]?.href
  }))

export const userSectionTabs = (group: UserNavGroup): KunTabItem[] =>
  group.sections.map((s) => ({
    value: s.value,
    textValue: typeof s.count === 'number' ? `${s.label} ${s.count}` : s.label,
    href: s.href
  }))
