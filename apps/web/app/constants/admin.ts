import type { ForumPermission } from '~/composables/useCan'

export const KUN_ADMIN_OVERVIEW_STATS_MODEL_ITEM = [
  'topic_count',
  'reply_count',
  'topic_comment_count',
  'work_count',
  'galgame_resource_count',
  'galgame_comment_count',
  'website_count',
  'website_comment_count',
  'direct_message_count'
] as const

export type StatsModelType =
  (typeof KUN_ADMIN_OVERVIEW_STATS_MODEL_ITEM)[number]

export interface ChartItem {
  label: string
  color: string
}

export const KUN_ADMIN_OVERVIEW_STATS_MODEL_MAP: Record<
  StatsModelType,
  ChartItem
> = {
  topic_count: { label: '话题', color: '#7828C8' },
  reply_count: { label: '话题回复', color: '#17C964' },
  topic_comment_count: { label: '话题评论', color: '#F31260' },
  work_count: { label: 'Galgame', color: '#FF4ECD' },
  galgame_resource_count: { label: 'Galgame 资源', color: '#F5A524' },
  galgame_comment_count: { label: 'Galgame 评论', color: '#7EE7FC' },
  website_count: { label: 'Galgame 网站', color: '#7ccf00' },
  website_comment_count: { label: 'Galgame 网站评论', color: '#ff637e' },
  direct_message_count: { label: '聊天消息', color: '#ff8904' }
} as const

export const KUN_ADMIN_PAGE_ROUTE = [
  'overview',
  'user',
  'submissions',
  'moderation',
  'topic',
  'friend-link',
  'website',
  'doc',
  'update',
  'permission',
  'setting'
]

export type KUN_ADMIN_PAGE_ROUTE_TYPE = (typeof KUN_ADMIN_PAGE_ROUTE)[number]

export interface KunAdminPageAsideItem {
  name: KUN_ADMIN_PAGE_ROUTE_TYPE
  label: string
  icon?: string
  router?: KUN_ADMIN_PAGE_ROUTE_TYPE
  permissions?: ForumPermission[]
  role?: 'admin'
  to?: string
}

export const KUN_ADMIN_PAGE_ASIDE_NAV_ITEM: KunAdminPageAsideItem[] = [
  {
    name: 'overview',
    label: '数据总览',
    icon: 'lucide:chart-area',
    router: 'overview',
    permissions: ['admin.dashboard']
  },
  {
    name: 'user',
    label: '用户管理',
    icon: 'lucide:user',
    router: 'user',
    permissions: ['user.purge_content']
  },
  {
    name: 'submissions',
    label: 'Galgame 审核',
    icon: 'lucide:clipboard-check',
    router: 'submissions',
    permissions: ['galgame.claim.review']
  },
  {
    name: 'moderation',
    label: '内容审核',
    icon: 'lucide:shield-alert',
    router: 'moderation',
    permissions: ['trust.review']
  },
  {
    name: 'topic',
    label: '隐藏话题管理',
    icon: 'lucide:eye-off',
    router: 'topic',
    permissions: ['topic.view_hidden']
  },
  {
    name: 'friend-link',
    label: '友链管理',
    icon: 'lucide:link',
    router: 'friend-link',
    permissions: [
      'friend_link.create',
      'friend_link.edit',
      'friend_link.delete'
    ]
  },
  {
    name: 'website',
    label: '网站资料库管理',
    icon: 'lucide:globe',
    router: 'website',
    permissions: ['website.create', 'website.edit', 'website.delete']
  },
  {
    name: 'doc',
    label: '文档管理',
    icon: 'lucide:file-text',
    router: 'doc',
    permissions: ['doc.create', 'doc.edit', 'doc.delete']
  },
  {
    name: 'update',
    label: '更新日志与待办',
    icon: 'lucide:list-checks',
    to: '/update/todo',
    permissions: ['update_log.create', 'update_log.edit', 'update_log.delete']
  },
  {
    name: 'permission',
    label: '权限管理',
    icon: 'lucide:shield-check',
    router: 'permission',
    role: 'admin'
  },
  {
    name: 'setting',
    label: '网站设置',
    icon: 'lucide:settings',
    router: 'setting',
    role: 'admin'
  }
]
