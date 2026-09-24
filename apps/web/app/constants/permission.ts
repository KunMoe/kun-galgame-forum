import type { ForumPermission } from '~/composables/useCan'

export interface KunPermissionMeta {
  label: string
  group: string
}

export const KUN_PERMISSION_GROUP_ORDER = [
  '话题',
  '回复',
  '评论',
  '投票',
  'Galgame',
  '收藏夹',
  '题目',
  '资源',
  '评分',
  '工具集',
  '文档',
  '网站',
  '友链',
  '更新日志与待办',
  '管理'
] as const

export const KUN_PERMISSION_META: Record<ForumPermission, KunPermissionMeta> = {
  'topic.edit_any': { label: '编辑任意话题', group: '话题' },
  'topic.hide': { label: '隐藏话题', group: '话题' },
  'topic.view_hidden': { label: '查看被隐藏的话题', group: '话题' },
  'topic.view_restricted': { label: '查看受限话题', group: '话题' },
  'topic.delete_any': { label: '彻底删除话题', group: '话题' },
  'topic.set_best_answer': { label: '设置最佳答案', group: '话题' },
  'reply.edit_any': { label: '编辑任意回复', group: '回复' },
  'reply.delete_any': { label: '删除任意回复', group: '回复' },
  'reply.pin': { label: '置顶回复', group: '回复' },
  'comment.topic.edit': { label: '编辑话题评论', group: '评论' },
  'comment.topic.delete': { label: '删除话题评论', group: '评论' },
  'comment.galgame.edit': { label: '编辑 Galgame 评论', group: '评论' },
  'comment.galgame.delete': { label: '删除 Galgame 评论', group: '评论' },
  'comment.rating.edit': { label: '编辑评分评论', group: '评论' },
  'comment.rating.delete': { label: '删除评分评论', group: '评论' },
  'comment.website.edit': { label: '编辑网站评论', group: '评论' },
  'comment.website.delete': { label: '删除网站评论', group: '评论' },
  'comment.toolset.edit': { label: '编辑工具集评论', group: '评论' },
  'comment.toolset.delete': { label: '删除工具集评论', group: '评论' },
  'comment.resource.edit': { label: '编辑资源评论', group: '评论' },
  'comment.resource.delete': { label: '删除资源评论', group: '评论' },
  'comment.quiz.edit': { label: '编辑题目评论', group: '评论' },
  'comment.quiz.delete': { label: '删除题目评论', group: '评论' },
  'poll.create_any': { label: '为任意话题创建投票', group: '投票' },
  'poll.edit_any': { label: '编辑任意投票', group: '投票' },
  'poll.delete_any': { label: '删除任意投票', group: '投票' },
  'poll.view_restricted': { label: '查看受限/匿名投票结果', group: '投票' },
  'lottery.create_any': { label: '为任意话题创建抽奖', group: '抽奖' },
  'lottery.manage_any': { label: '管理任意抽奖', group: '抽奖' },
  'lottery.view_restricted': { label: '查看隐藏的抽奖参与名单', group: '抽奖' },
  'galgame.ban_resource_publish': {
    label: '禁止游戏资源发布',
    group: 'Galgame'
  },
  'galgame.claim.review': {
    label: '审核 Galgame 投稿',
    group: 'Galgame'
  },
  'galgame.edit_proposal.review': {
    label: '审核资料编辑提案',
    group: 'Galgame'
  },
  'collection.edit_any': { label: '编辑任意收藏夹', group: '收藏夹' },
  'collection.delete_any': { label: '删除任意收藏夹', group: '收藏夹' },
  'quiz.edit_any': { label: '编辑任意题目', group: '题目' },
  'quiz.delete_any': { label: '删除任意题目', group: '题目' },
  'resource.edit_any': { label: '编辑任意游戏资源', group: '资源' },
  'resource.delete_any': { label: '删除任意游戏资源', group: '资源' },
  'rating.delete_any': { label: '删除任意评分', group: '评分' },
  'toolset.edit_any': { label: '编辑任意工具集', group: '工具集' },
  'toolset.delete_any': { label: '删除任意工具集', group: '工具集' },
  'toolset.resource.edit_any': {
    label: '编辑任意工具集资源',
    group: '工具集'
  },
  'toolset.resource.delete_any': {
    label: '删除任意工具集资源',
    group: '工具集'
  },
  'toolset.upload_bypass': { label: '向任意工具集上传', group: '工具集' },
  'doc.create': { label: '创建文档', group: '文档' },
  'doc.edit': { label: '编辑文档', group: '文档' },
  'doc.delete': { label: '删除文档', group: '文档' },
  'website.create': { label: '创建网站', group: '网站' },
  'website.edit': { label: '编辑网站', group: '网站' },
  'website.delete': { label: '删除网站', group: '网站' },
  'friend_link.create': { label: '创建友链', group: '友链' },
  'friend_link.edit': { label: '编辑友链', group: '友链' },
  'friend_link.delete': { label: '删除友链', group: '友链' },
  'update_log.create': { label: '创建更新日志与待办', group: '更新日志与待办' },
  'update_log.edit': { label: '编辑更新日志与待办', group: '更新日志与待办' },
  'update_log.delete': { label: '删除更新日志与待办', group: '更新日志与待办' },
  'update_log.reopen': { label: '重新启用已废弃待办', group: '更新日志与待办' },
  'trust.review': { label: '处理举报与内容审核', group: '管理' },
  'admin.dashboard': { label: '管理总览与统计', group: '管理' },
  'user.purge_content': { label: '清除用户全部内容', group: '管理' }
}

export const KUN_PERMISSION_KEYS = Object.keys(
  KUN_PERMISSION_META
) as ForumPermission[]

export const KUN_PERMISSION_GROUPS: {
  group: string
  perms: ForumPermission[]
}[] = KUN_PERMISSION_GROUP_ORDER.map((group) => ({
  group,
  perms: KUN_PERMISSION_KEYS.filter(
    (key) => KUN_PERMISSION_META[key].group === group
  )
})).filter((entry) => entry.perms.length)

export interface KunPermRoleColumn {
  role: string
  label: string
  editable: boolean
  locked: boolean
}

export const KUN_PERM_ROLE_COLUMNS: KunPermRoleColumn[] = [
  { role: 'creator', label: '创作者', editable: true, locked: false },
  { role: 'moderator', label: '版主', editable: true, locked: false },
  { role: 'admin', label: '管理员', editable: true, locked: false },
  { role: 'ren', label: '莲', editable: false, locked: true }
]

export const KUN_PERM_EDITABLE_ROLES = [
  'creator',
  'moderator',
  'admin'
] as const

export interface KunProxyPermission {
  key: string
  label: string
  note: string
}

export const KUN_PROXY_PERMISSIONS: KunProxyPermission[] = [
  {
    key: 'galgame.create',
    label: 'Galgame 直发建条目',
    note: '版主+，绕过提交队列；镜像 NextMoe galgame.create'
  },
  {
    key: 'galgame.review_submission',
    label: 'Galgame 提交审核队列',
    note: '版主+；纯查看门（裁决权归 infra catalog.claim.review，按用户 token 判定）'
  },
  {
    key: 'galgame.review',
    label: 'Galgame 提案查看 / 队列',
    note: '版主+ 或条目创建者；纯查看门（裁决权归 infra，按用户 token 判定）'
  },
  {
    key: 'taxonomy.edit',
    label: '资料库条目编辑',
    note: '莲裁定的更严门槛（仅 admin / ren，NextMoe 为版主+）'
  },
  {
    key: 'taxonomy.delete',
    label: '资料库条目删除',
    note: '莲裁定的更严门槛（仅 admin / ren，NextMoe 为版主+）'
  },
  {
    key: 'taxonomy.revert',
    label: '资料库条目回滚',
    note: '莲裁定的更严门槛（仅 admin / ren，NextMoe 为版主+）'
  },
  {
    key: 'trust.queue_access',
    label: 'Trust 举报收件箱',
    note: '版主+；镜像 NextMoe trust.queue_access'
  }
]
