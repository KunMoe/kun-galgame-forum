import type {
  TopicAccessRole,
  TopicAccessScope,
  TopicCategoryKey
} from '~/constants/topic'

export const useTopicEditorStore = () => {
  const tempStore = useTempEditStore()
  const persistStore = usePersistEditTopicStore()

  const activeStore = computed(() =>
    tempStore.isTopicRewriting ? tempStore : persistStore
  )

  const category = computed<TopicCategoryKey | ''>({
    get: () => activeStore.value.category as TopicCategoryKey | '',
    set: (value) => {
      activeStore.value.category = value
    }
  })

  const section = computed<string[]>({
    get: () => activeStore.value.section,
    set: (value) => {
      activeStore.value.section = value
    }
  })

  const title = computed<string>({
    get: () => activeStore.value.title,
    set: (value) => {
      activeStore.value.title = value
    }
  })

  const content = computed<string>({
    get: () => activeStore.value.content,
    set: (value) => {
      activeStore.value.content = value
    }
  })

  const isNSFW = computed<boolean>({
    get: () => activeStore.value.isNSFW,
    set: (value) => {
      activeStore.value.isNSFW = value
    }
  })

  const coverImages = computed<string[]>({
    get: () => activeStore.value.coverImages,
    set: (value) => {
      activeStore.value.coverImages = value
    }
  })

  const accessScope = computed<TopicAccessScope>({
    get: () => activeStore.value.accessScope,
    set: (value) => {
      activeStore.value.accessScope = value
    }
  })

  const accessRoles = computed<TopicAccessRole[]>({
    get: () => activeStore.value.accessRoles,
    set: (value) => {
      activeStore.value.accessRoles = value
    }
  })

  const accessUserIds = computed<number[]>({
    get: () => activeStore.value.accessUserIds,
    set: (value) => {
      activeStore.value.accessUserIds = value
    }
  })

  // A topic being edited manages its polls and lotteries on its own page.
  const { polls, lotteries } = storeToRefs(persistStore)

  return {
    category,
    section,
    title,
    content,
    isNSFW,
    coverImages,
    accessScope,
    accessRoles,
    accessUserIds,
    polls,
    lotteries
  }
}
