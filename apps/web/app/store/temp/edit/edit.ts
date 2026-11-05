import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { EditStoreTemp } from '~/store/types/edit/topic'

export const useTempEditStore = defineStore(
  'tempEdit',
  () => {
    const id = ref<EditStoreTemp['id']>(0)
    const title = ref<EditStoreTemp['title']>('')
    const content = ref<EditStoreTemp['content']>('')
    const section = ref<EditStoreTemp['section']>([])
    const isNSFW = ref<EditStoreTemp['isNSFW']>(false)
    const isEntityCoverBlurred = ref<EditStoreTemp['isEntityCoverBlurred']>(false)
    const coverImages = ref<EditStoreTemp['coverImages']>([])
    const accessScope = ref<EditStoreTemp['accessScope']>('public')
    const accessRoles = ref<EditStoreTemp['accessRoles']>([])
    const accessUserIds = ref<EditStoreTemp['accessUserIds']>([])
    const accessUsers = ref<EditStoreTemp['accessUsers']>([])
    const isTopicRewriting = ref<EditStoreTemp['isTopicRewriting']>(false)

    const resetRewriteTopicData = () => {
      id.value = 0
      title.value = ''
      content.value = ''
      section.value = []
      isNSFW.value = false
      isEntityCoverBlurred.value = false
      coverImages.value = []
      accessScope.value = 'public'
      accessRoles.value = []
      accessUserIds.value = []
      accessUsers.value = []
      isTopicRewriting.value = false
    }

    return {
      id,
      title,
      content,
      section,
      isNSFW,
      isEntityCoverBlurred,
      coverImages,
      accessScope,
      accessRoles,
      accessUserIds,
      accessUsers,
      isTopicRewriting,
      resetRewriteTopicData
    }
  },
  {
    persist: false
  }
)
