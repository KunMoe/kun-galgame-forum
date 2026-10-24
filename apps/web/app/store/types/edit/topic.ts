import type { TopicAccessRole, TopicAccessScope } from '~/constants/topic'
import type { PollFormData } from '~/components/topic/poll/types'
import type { LotteryFormData } from '~/components/topic/lottery/types'

export interface EditStorePersist {
  mode: 'preview' | 'code'

  title: string
  content: string
  section: string[]
  isNSFW: boolean
  coverImages: string[]
  accessScope: TopicAccessScope
  accessRoles: TopicAccessRole[]
  accessUserIds: number[]
  polls: PollFormData[]
  lotteries: LotteryFormData[]
}

export interface EditStoreTemp {
  id: number
  title: string
  content: string
  section: string[]
  isNSFW: boolean
  coverImages: string[]
  accessScope: TopicAccessScope
  accessRoles: TopicAccessRole[]
  accessUserIds: number[]
  accessUsers: KunUser[]

  isTopicRewriting: boolean
}
