type ChipColor = 'warning' | 'primary' | 'danger' | 'default'

export const TRUST_REVIEW_STATUS: Record<
  number,
  { label: string; color: ChipColor }
> = {
  0: { label: '待处理', color: 'warning' },
  1: { label: '处理中', color: 'primary' },
  2: { label: '已处置', color: 'danger' },
  3: { label: '已驳回', color: 'default' }
}

export const TRUST_REVIEW_SOURCE: Record<number, string> = {
  0: '用户举报',
  1: 'AI 文本',
  2: 'AI 图片',
  3: '社区转入',
  4: '分级纠错',
  5: '人工新建',
  6: 'AI 抽样校准'
}

export const TRUST_ACTIONS = [
  { value: 1, label: '隐藏内容' },
  { value: 2, label: '删除内容' },
  { value: 3, label: '警告作者' },
  { value: 4, label: '限制用户' },
  { value: 5, label: '升级至账号中心' },
  { value: 0, label: '不处置（仅记录）' }
] as const

export const TRUST_SUBJECT_KIND: Record<string, string> = {
  forum_topic: '话题',
  forum_reply: '回复',
  forum_comment: '话题评论',
  galgame: 'Galgame',
  galgame_comment: '游戏评论',
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
