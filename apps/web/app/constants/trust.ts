import type {
  ReviewItemPatch,
  ReviewItemSummary
} from '#shared/utils/api/schemas'

type ChipColor = 'warning' | 'primary' | 'danger' | 'default'

export type ReviewState = ReviewItemSummary['state']
export type DispositionAction = NonNullable<ReviewItemPatch['action']>

export const TRUST_REVIEW_STATE: Record<
  ReviewState,
  { label: string; color: ChipColor }
> = {
  pending: { label: '待处理', color: 'warning' },
  claimed: { label: '处理中', color: 'primary' },
  actioned: { label: '已处置', color: 'danger' },
  dismissed: { label: '已驳回', color: 'default' }
}

export const TRUST_REVIEW_ORIGIN: Record<string, string> = {
  reports: '用户举报',
  ai_text: 'AI 文本',
  ai_image: 'AI 图片',
  community_forward: '社区转入',
  mislabel: '分级纠错',
  manual: '人工新建',
  ai_sample: 'AI 抽样校准'
}

export const TRUST_ACTIONS: { value: DispositionAction; label: string }[] = [
  { value: 'hide', label: '隐藏内容' },
  { value: 'remove', label: '删除内容' },
  { value: 'warn_user', label: '警告作者' },
  { value: 'restrict', label: '限制用户' },
  { value: 'escalate_idp', label: '升级至账号中心' },
  { value: 'none', label: '不处置（仅记录）' }
]

export const TRUST_SUBJECT_KIND: Record<string, string> = {
  forum_topic: '话题',
  forum_reply: '回复',
  forum_comment: '话题评论',
  forum_topic_poll: '投票',
  forum_topic_lottery: '抽奖',
  forum_todo: '待办',
  galgame: 'Galgame',
  galgame_comment: '游戏评论',
  galgame_rating: '游戏评价',
  galgame_resource: '游戏资源',
  galgame_collection: '收藏夹',
  galgame_quiz: '游戏题目',
  galgame_quiz_answer: '题目作答',
  galgame_toolset: '工具',
  galgame_toolset_resource: '工具资源',
  community_post: '社区评论',
  user: '用户'
}

export const trustSubjectHref = (
  kind: string,
  id: string
): string | undefined => {
  switch (kind) {
    case 'forum_topic':
      return `/topic/${id}`
    case 'galgame':
      return `/galgame/${id}`
    case 'user':
      return `/user/${id}`
    default:
      return undefined
  }
}
