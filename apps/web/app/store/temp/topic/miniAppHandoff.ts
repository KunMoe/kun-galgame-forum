import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { PollFormData } from '~/components/topic/poll/types'
import type { LotteryFormData } from '~/components/topic/lottery/types'

export type HeldMiniApp =
  | { key: 'poll'; form: PollFormData }
  | { key: 'lottery'; form: LotteryFormData }

export const useTempTopicMiniAppHandoffStore = defineStore(
  'tempTopicMiniAppHandoff',
  () => {
    const topicId = ref('')
    const held = ref<HeldMiniApp[]>([])

    const hold = (id: string, items: HeldMiniApp[]) => {
      topicId.value = id
      held.value = items
    }

    const heldFor = (id: string) => (topicId.value === id ? held.value : [])

    const release = (item: HeldMiniApp) => {
      held.value = held.value.filter((entry) => entry !== item)
    }

    return { topicId, held, hold, heldFor, release }
  },
  {
    persist: false
  }
)
