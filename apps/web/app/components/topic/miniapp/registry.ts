import type { KunUIColor } from '@kungal/ui-core'

export type TopicMiniAppKey = 'poll' | 'lottery'

export interface TopicMiniApp {
  key: TopicMiniAppKey
  label: string
  tagline: string
  icon: string
  badge: string
  badgeColor: KunUIColor
  help: string[]
}

// The 话题小程序 panel used to hardcode its one button and its help text inside
// the poll component, so the second mini-app could only be added by editing the
// first one. Everything the panel renders now comes from here.
export const TOPIC_MINI_APPS: TopicMiniApp[] = [
  {
    key: 'poll',
    label: '投票',
    tagline: '让读者在几个选项里表态, 结果实时统计',
    icon: 'lucide:bar-chart-3',
    badge: '投票话题',
    badgeColor: 'primary',
    help: [
      '让读者在若干选项里表态, 结果实时统计。',
      '可以设置单选/多选、截止时间、匿名与否, 以及结果什么时候公开。',
      '每个话题最多 30 个投票, 每个投票最多 20 个选项。'
    ]
  },
  {
    key: 'lottery',
    label: '抽奖',
    tagline: '送码、送周边或送萌萌点, 由系统产生中奖名单',
    icon: 'lucide:gift',
    badge: '抽奖话题',
    badgeColor: 'secondary',
    help: [
      '送激活码、实物周边或萌萌点, 由系统产生中奖名单。',
      '开奖前公示随机数承诺 (哈希), 开奖后公布随机数, 任何人都能自己验算一遍。',
      '楼层抽奖不使用随机数, 中奖楼层数出来就是, 读者数楼即可核对。',
      '激活码由系统加密托管, 只有中奖者本人能查看; 实物请与中奖者私聊沟通。萌萌点奖池由发起人出资, 未发出的部分会退回。',
      '兑换码需在开奖后 7 天内领取, 逾期自动作废。',
      '本站只负责产生名单, 不担保实物履约, 请自行与对方确认。'
    ]
  }
]

// The list surfaces get bare kind strings from the API (`mini_apps`), matching
// pkg/miniapp on the Go side.
export const topicMiniAppsOf = (keys?: string[]): TopicMiniApp[] =>
  (keys ?? [])
    .map((key) => TOPIC_MINI_APPS.find((app) => app.key === key))
    .filter((app): app is TopicMiniApp => !!app)
