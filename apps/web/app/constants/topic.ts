import type { KunCheckBoxGroupOption, KunSelectOption } from '@kungal/ui-vue'
import { KUN_USER_ROLE_MAP } from '~/constants/user'

export const KUN_TOPIC_CATEGORY: Record<string, string> = {
  galgame: 'Galgame',
  technique: '技术交流',
  others: '其它话题'
}

export const KUN_TOPIC_SECTION: Record<string, string> = {
  'g-walkthrough': '攻略',
  'g-chatting': '闲聊',
  'g-article': '文章',
  'g-seeking': '寻求资源',
  'g-news': '资讯',
  'g-releases': '新作消息',
  'g-other': '其它',
  't-crack': '逆向工程',
  't-web': 'Web',
  't-languages': '编程语言',
  't-help': '请求帮助',
  't-linux': 'Linux',
  't-practical': '实用技术',
  't-ai': 'AI',
  't-android': 'Android',
  't-adobe': 'Adobe',
  't-algorithm': '算法',
  't-other': '其它',
  'o-anime': '动漫',
  'o-comics': '漫画',
  'o-music': '音乐',
  'o-novel': '轻小说',
  'o-daily': '日常',
  'o-essay': '个人随笔',
  'o-forum': '论坛相关',
  'o-patch': '补丁网站',
  'o-other': '其它'
}

export const KUN_TOPIC_SECTION_CONST = [
  'g-walkthrough',
  'g-chatting',
  'g-article',
  'g-seeking',
  'g-news',
  'g-releases',
  'g-other',
  't-crack',
  't-web',
  't-languages',
  't-help',
  't-linux',
  't-practical',
  't-ai',
  't-android',
  't-adobe',
  't-algorithm',
  't-other',
  'o-anime',
  'o-comics',
  'o-music',
  'o-novel',
  'o-daily',
  'o-essay',
  'o-forum',
  'o-patch',
  'o-other'
] as const

export const TOPIC_SORT_FIELD_CONST = [
  'created',
  'view',
  'view_1d',
  'view_7d',
  'view_30d',
  'status_update_time',
  'like',
  'favorite',
  'upvote'
] as const

export const TOPIC_CATEGORIES = {
  galgame: {
    key: 'galgame',
    label: 'Galgame',
    icon: 'lucide:gamepad-2'
  },
  technique: {
    key: 'technique',
    label: '技术交流',
    icon: 'lucide:drafting-compass'
  },
  others: {
    key: 'others',
    label: '其它话题',
    icon: 'lucide:circle-ellipsis'
  }
} as const

export type TopicCategoryKey = keyof typeof TOPIC_CATEGORIES

export const TOPIC_SECTIONS: Record<
  TopicCategoryKey,
  Record<string, string>
> = {
  galgame: {
    'g-walkthrough': '攻略',
    'g-chatting': '闲聊',
    'g-article': '文章',
    'g-seeking': '寻求资源',
    'g-news': '资讯',
    'g-releases': '新作消息',
    'g-other': '其它'
  },
  technique: {
    't-crack': '逆向工程',
    't-web': 'Web',
    't-languages': '编程语言',
    't-help': '请求帮助',
    't-linux': 'Linux',
    't-practical': '实用技术',
    't-ai': 'AI',
    't-android': 'Android',
    't-adobe': 'Adobe',
    't-algorithm': '算法',
    't-other': '其它'
  },
  others: {
    'o-anime': '动漫',
    'o-comics': '漫画',
    'o-music': '音乐',
    'o-novel': '轻小说',
    'o-daily': '日常',
    'o-essay': '个人随笔',
    'o-forum': '论坛相关',
    'o-patch': '补丁网站',
    'o-other': '其它'
  }
}

export const TOPIC_POLL_VISIBILITY_OPTIONS = [
  { value: 'always', label: '任何人可见结果' },
  { value: 'after_vote', label: '投票后可见结果' },
  { value: 'after_deadline', label: '结束后可见结果' }
] as const

export const TOPIC_POLL_VISIBILITY_MAP: Record<string, string> = {
  always: '任何人可见结果',
  after_vote: '投票后可见结果',
  after_deadline: '结束后可见结果'
}

type TopicHiddenByColor = 'default' | 'warning' | 'danger'

export interface TopicHiddenByMeta {
  label: string
  notice: string
  color: TopicHiddenByColor
}

export const KUN_TOPIC_HIDDEN_BY_FALLBACK: TopicHiddenByMeta = {
  label: '已隐藏',
  notice: '该话题已被隐藏',
  color: 'default'
}

export const KUN_TOPIC_HIDDEN_BY: Record<string, TopicHiddenByMeta> = {
  author: {
    label: '作者隐藏',
    notice: '该话题已被作者隐藏',
    color: 'warning'
  },
  moderator: {
    label: '管理员隐藏',
    notice: '该话题已被管理员隐藏',
    color: 'danger'
  },
  trust: {
    label: '风纪隐藏',
    notice: '该话题已因违规处理被隐藏',
    color: 'danger'
  }
}

export const topicHiddenByMeta = (hiddenBy: string): TopicHiddenByMeta =>
  KUN_TOPIC_HIDDEN_BY[hiddenBy] ?? KUN_TOPIC_HIDDEN_BY_FALLBACK

export const KUN_TOPIC_ACCESS_SCOPE_CONST = [
  'public',
  'login',
  'role',
  'users'
] as const

export type TopicAccessScope = (typeof KUN_TOPIC_ACCESS_SCOPE_CONST)[number]

export interface TopicAccessScopeMeta {
  label: string
  icon: string
  hint: string
  notice: string
  description: string
}

export const KUN_TOPIC_ACCESS_SCOPE_FALLBACK: TopicAccessScopeMeta = {
  label: '受限',
  icon: 'lucide:lock',
  hint: '这个话题的访问范围受到限制',
  notice: '本话题的访问范围受限',
  description: '只有符合条件的用户可以打开它'
}

export const KUN_TOPIC_ACCESS_SCOPE: Record<string, TopicAccessScopeMeta> = {
  public: {
    label: '公开',
    icon: 'lucide:globe',
    hint: '所有人都能打开这个话题, 它会正常出现在列表, 搜索与首页动态中',
    notice: '本话题公开可见',
    description: '所有人都能打开它'
  },
  login: {
    label: '登录用户可见',
    icon: 'lucide:log-in',
    hint: '未登录的访客打不开这个话题。它仍然会出现在已登录用户的列表与搜索中, 但不会进入首页动态, 排行榜与 RSS',
    notice: '本话题仅登录用户可见',
    description: '未登录的访客打不开它, 它也不会出现在首页动态, 排行榜与 RSS 中'
  },
  role: {
    label: '指定角色可见',
    icon: 'lucide:shield',
    hint: '只有持有选中角色的用户能打开这个话题 (您自己与管理人员始终可以打开)。它不会出现在任何列表与搜索中, 只能通过链接访问',
    notice: '本话题仅指定角色可见',
    description: '它不会出现在任何列表与搜索中, 只能通过链接访问'
  },
  users: {
    label: '指定用户可见',
    icon: 'lucide:users',
    hint: '只有被指定的用户能打开这个话题 (您自己与管理人员始终可以打开)。它不会出现在任何列表与搜索中, 只能通过链接访问',
    notice: '本话题仅指定用户可见',
    description: '它不会出现在任何列表与搜索中, 只能通过链接访问'
  }
}

export const topicAccessScopeMeta = (scope: string): TopicAccessScopeMeta =>
  KUN_TOPIC_ACCESS_SCOPE[scope] ?? KUN_TOPIC_ACCESS_SCOPE_FALLBACK

export const KUN_TOPIC_ACCESS_SCOPE_OPTIONS: KunSelectOption<TopicAccessScope>[] =
  KUN_TOPIC_ACCESS_SCOPE_CONST.map((value) => ({
    value,
    label: topicAccessScopeMeta(value).label
  }))

// `user` is missing on purpose. The API accepts it as a grant subject, but OAuth
// never puts it in the roles claim (docs/oauth/11-roles.md §2 — it is the
// implicit default identity), so a `user` grant matches nobody and the topic
// silently becomes invisible to everyone the author picked. The login scope is
// what "any signed-in user" means here.
export const KUN_TOPIC_ACCESS_ROLE_CONST = [
  'creator',
  'moderator',
  'admin',
  'ren'
] as const

export type TopicAccessRole = (typeof KUN_TOPIC_ACCESS_ROLE_CONST)[number]

export const KUN_TOPIC_ACCESS_ROLE_OPTIONS: KunCheckBoxGroupOption<TopicAccessRole>[] =
  KUN_TOPIC_ACCESS_ROLE_CONST.map((value) => ({
    value,
    label: KUN_USER_ROLE_MAP[value] ?? value
  }))

export const KUN_TOPIC_ACCESS_ROLE_LIMIT = 8
export const KUN_TOPIC_ACCESS_USER_LIMIT = 50

export const toTopicAccessScope = (scope?: string): TopicAccessScope =>
  KUN_TOPIC_ACCESS_SCOPE_CONST.find((value) => value === scope) ?? 'public'

export const toTopicAccessRoles = (roles?: string[]): TopicAccessRole[] =>
  (roles ?? []).filter((role): role is TopicAccessRole =>
    KUN_TOPIC_ACCESS_ROLE_CONST.some((value) => value === role)
  )

export const KUN_LOTTERY_ENTRY_MODE: Record<string, string> = {
  signup: '报名参与',
  reply: '回帖后参与',
  floor: '楼层抽奖'
}

export const KUN_LOTTERY_ENTRY_MODE_OPTIONS = [
  { value: 'signup', label: '报名参与 — 点一下就参加' },
  { value: 'reply', label: '回帖后参与 — 需要先在本话题回复' },
  { value: 'floor', label: '楼层抽奖 — 由楼层号直接决定, 无需报名' }
] as const

export const KUN_LOTTERY_DRAW_MODE: Record<string, string> = {
  deadline: '到点自动开奖',
  manual: '楼主手动开奖',
  threshold: '满员自动开奖'
}

export const KUN_LOTTERY_DRAW_MODE_OPTIONS = [
  { value: 'deadline', label: '到点自动开奖' },
  { value: 'manual', label: '楼主手动开奖' },
  { value: 'threshold', label: '满员自动开奖' }
] as const

export const KUN_LOTTERY_DELIVERY: Record<string, string> = {
  code: '系统托管兑换码',
  offline: '楼主私聊发放',
  point: '自动发放萌萌点'
}

export const KUN_LOTTERY_DELIVERY_OPTIONS = [
  { value: 'offline', label: '楼主私聊发放 (实物周边等)' },
  { value: 'code', label: '系统托管兑换码 (激活码等)' },
  { value: 'point', label: '自动发放萌萌点' }
] as const

export const KUN_LOTTERY_POINT_MODE_OPTIONS = [
  { value: 'fixed', label: '每人固定 — 每位中奖者都拿同样多' },
  { value: 'split', label: '奖池均分 — 总额平均分给中奖者' },
  { value: 'random', label: '奖池拼手气 — 总额随机分, 每人至少 1 点' }
] as const

export const KUN_LOTTERY_STATUS: Record<string, string> = {
  open: '进行中',
  drawing: '开奖中',
  drawn: '已开奖',
  cancelled: '已取消'
}

export const KUN_LOTTERY_ENTER_BLOCKED: Record<string, string> = {
  not_open: '抽奖已经结束',
  past_closes_at: '抽奖已过截止时间',
  no_signup: '楼层抽奖无需报名, 直接回帖即可',
  own_lottery: '不能参加自己发起的抽奖',
  reply_required: '本抽奖要求先在该话题下回复',
  moemoepoint_below_minimum: '萌萌点未达到本抽奖的门槛',
  account_too_new: '注册时间未达到本抽奖的门槛'
}

export const KUN_LOTTERY_FULFILLMENT: Record<string, string> = {
  pending: '待发放',
  shipped: '已发出',
  received: '已确认收到',
  forfeited: '已放弃'
}
