import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { EditStorePersist } from '~/store/types/edit/topic'
import { lotteryWithoutCodes } from '~/composables/topic/pendingMiniApps'

export const usePersistEditTopicStore = defineStore(
  'KUNGalgameEditTopic',
  () => {
    const mode = ref<EditStorePersist['mode']>('preview')
    const title = ref<EditStorePersist['title']>('')
    const content = ref<EditStorePersist['content']>('')
    const category = ref<EditStorePersist['category']>('')
    const section = ref<EditStorePersist['section']>([])
    const isNSFW = ref<EditStorePersist['isNSFW']>(false)
    const coverImages = ref<EditStorePersist['coverImages']>([])
    const accessScope = ref<EditStorePersist['accessScope']>('public')
    const accessRoles = ref<EditStorePersist['accessRoles']>([])
    const accessUserIds = ref<EditStorePersist['accessUserIds']>([])
    const polls = ref<EditStorePersist['polls']>([])
    const lotteries = ref<EditStorePersist['lotteries']>([])

    const resetTopicData = () => {
      title.value = ''
      content.value = ''
      category.value = ''
      section.value = []
      isNSFW.value = false
      coverImages.value = []
      accessScope.value = 'public'
      accessRoles.value = []
      accessUserIds.value = []
      polls.value = []
      lotteries.value = []
    }

    return {
      mode,
      title,
      content,
      category,
      section,
      isNSFW,
      coverImages,
      accessScope,
      accessRoles,
      accessUserIds,
      polls,
      lotteries,
      resetTopicData
    }
  },
  {
    persist: {
      storage: piniaPluginPersistedstate.localStorage(),
      // Redemption codes stay in memory only: once submitted they are sealed
      // and never shown again, so they must not sit in localStorage in clear.
      serializer: {
        serialize: (state) =>
          JSON.stringify({
            ...state,
            lotteries: state.lotteries.map(lotteryWithoutCodes)
          }),
        deserialize: JSON.parse
      }
    }
  }
)
