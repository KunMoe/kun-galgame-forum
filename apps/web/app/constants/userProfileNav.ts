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

export type UserNavItem = {
  value: string
  label: string
  icon: string
  sections: UserNavSection[]
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
      sections: [section('activity', '动态', 'activity')]
    },
    {
      value: 'library',
      label: '游戏库',
      icon: 'lucide:library',
      sections: [section('library', '游戏库', 'library')]
    },
    {
      value: 'collection',
      label: '收藏',
      icon: 'lucide:heart',
      sections: [
        section('collection-galgame', '游戏收藏夹', 'collection/galgame'),
        section('collection-topic', '话题收藏', 'collection/topic'),
        section('collection-toolset', '工具收藏', 'collection/toolset')
      ]
    },
    {
      value: 'follow',
      label: '关注',
      icon: 'lucide:users',
      sections: [
        section('following', '关注', 'follow/following', {
          count: c.following_count
        }),
        section('followers', '粉丝', 'follow/followers', {
          count: c.follower_count
        }),
        section('followed-companies', '会社', 'followed-companies')
      ]
    },
    {
      value: 'publish',
      label: '发布',
      icon: 'lucide:square-pen',
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
        }),
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
    },
    {
      value: 'info',
      label: '关于',
      icon: 'lucide:user-round',
      sections: [section('info', '关于', 'info')]
    }
  ]

  if (isOwner) {
    items.push({
      value: 'setting',
      label: '设置',
      icon: 'lucide:settings',
      sections: [section('setting', '设置', 'setting')]
    })
  }

  if (isOwner) {
    return items
  }
  return items.map((item) => ({
    ...item,
    sections: item.sections.filter((s) => !(s.hideWhenEmpty && s.count === 0))
  }))
}

const matches = (path: string, prefix: string) =>
  path === prefix || path.startsWith(`${prefix}/`)

export const findUserNavSection = (nav: UserNavItem[], path: string) => {
  for (const item of nav) {
    const hit = item.sections.find((s) => matches(path, s.match))
    if (hit) {
      return { item, section: hit }
    }
  }
  return null
}

export const userNavTabs = (nav: UserNavItem[]): KunTabItem[] =>
  nav.map((item) => ({
    value: item.value,
    textValue: item.label,
    icon: item.icon,
    href: item.sections[0]?.href
  }))

export const userSectionTabs = (item: UserNavItem): KunTabItem[] =>
  item.sections.map((s) => ({
    value: s.value,
    textValue: typeof s.count === 'number' ? `${s.label} ${s.count}` : s.label,
    href: s.href
  }))
