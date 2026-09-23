<script setup lang="ts">
const props = defineProps<{
  workId: number
  targetUserId?: number
  favoriteCount: number
  isFavorited: boolean
}>()

const emits = defineEmits<{
  saved: [payload: { favorited: boolean }]
}>()

const { id } = usePersistUserStore()

const isFavorited = ref(props.isFavorited)
const favoriteCount = ref(props.favoriteCount)

watch(
  () => props.isFavorited,
  (value) => (isFavorited.value = value)
)
watch(
  () => props.favoriteCount,
  (value) => (favoriteCount.value = value)
)

const pickerOpen = ref(false)

const onClick = () => {
  if (!id) {
    useAuthModal().open()
    return
  }
  pickerOpen.value = true
}

const onSaved = (payload: { favorited: boolean }) => {
  if (isFavorited.value !== payload.favorited) {
    favoriteCount.value += payload.favorited ? 1 : -1
  }
  isFavorited.value = payload.favorited
  useMyGalgameInteractions().setFavorited(props.workId, payload.favorited)
  emits('saved', payload)
}
</script>

<template>
  <KunTooltip text="收藏">
    <span class="flex">
      <KunReaction
        :model-value="isFavorited"
        :count="favoriteCount"
        :toggle="false"
        icon="lucide:heart"
        color="danger"
        label="收藏"
        @click="onClick"
      />
    </span>
  </KunTooltip>

  <GalgameCollectionPickerModal
    v-model="pickerOpen"
    :work-id="workId"
    @saved="onSaved"
  />
</template>
