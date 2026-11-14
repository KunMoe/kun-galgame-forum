import type { UserActivityGroup } from '#shared/utils/api/schemas'

type Verb = UserActivityGroup['verb']

export const TIMELINE_VERBS = [
  'all',
  'publish',
  'reply',
  'comment',
  'rate',
  'like',
  'edit'
] as const

export type TimelineFilter = (typeof TIMELINE_VERBS)[number]

const FILTER_LABEL: Record<TimelineFilter, string> = {
  all: '全部',
  publish: '发布',
  reply: '回复',
  comment: '评论',
  rate: '评分',
  like: '推荐',
  edit: '编辑'
}

const VERB_ICON: Record<Verb, string> = {
  publish: 'lucide:square-pen',
  reply: 'lucide:message-square-reply',
  comment: 'lucide:message-circle',
  rate: 'lucide:star',
  like: 'lucide:thumbs-up',
  edit: 'lucide:pencil-line'
}

export const timelineFilterTabs = () =>
  TIMELINE_VERBS.map((value) => ({ value, textValue: FILTER_LABEL[value] }))

export const timelineVerbIcon = (verb: Verb) => VERB_ICON[verb]

const WEEKDAY = '日一二三四五六'

export const timelineDayLabel = (date: string, newestYear: string) => {
  const [year, month, day] = date.split('-')
  const weekday = WEEKDAY[new Date(`${date}T00:00:00Z`).getUTCDay()]
  const label = `${Number(month)} 月 ${Number(day)} 日 周${weekday}`
  return year === newestYear ? label : `${year} 年 ${label}`
}

export const groupTimelineByDay = (groups: UserActivityGroup[]) => {
  const days: { date: string; groups: UserActivityGroup[] }[] = []
  for (const group of groups) {
    const last = days.at(-1)
    if (last?.date === group.calendar_date) {
      last.groups.push(group)
    } else {
      days.push({ date: group.calendar_date, groups: [group] })
    }
  }
  return days
}
