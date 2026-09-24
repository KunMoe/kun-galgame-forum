<script setup lang="ts">
import { ref } from 'vue'
import type { Lottery } from '#shared/utils/api/schemas'

const props = defineProps<{
  topicId: number
  isTopicAdmin: boolean
}>()

const isCreateOpen = defineModel<boolean>('isCreateOpen', { default: false })

const topicId = computed(() => String(props.topicId))
const { allowsNsfw, stanceKey } = useContentStance()
const isModalOpen = ref(false)
const lotteryToEdit = ref<Lottery | undefined>(undefined)

const { data, refresh } = await useApi(
  () => `topic-lotteries:${topicId.value}:${stanceKey.value}`,
  (api, { signal }) =>
    api.GET('/topics/{topic_id}/lotteries', {
      params: {
        path: { topic_id: topicId.value },
        query: { include_nsfw: allowsNsfw.value }
      },
      signal
    })
)

const lotteries = computed(() => data.value?.items ?? [])

watch(isCreateOpen, (open) => {
  if (!open) {
    return
  }
  lotteryToEdit.value = undefined
  isModalOpen.value = true
  isCreateOpen.value = false
})

const openEditModal = (lottery: Lottery) => {
  lotteryToEdit.value = lottery
  isModalOpen.value = true
}
</script>

<template>
  <div class="space-y-3">
    <TopicLotteryCard
      v-for="lottery in lotteries"
      :key="lottery.id"
      :lottery="lottery"
      @edit="openEditModal"
      @refresh="refresh"
    />

    <TopicLotteryModal
      v-if="isTopicAdmin || lotteryToEdit"
      v-model="isModalOpen"
      :topic-id="topicId"
      :initial-data="lotteryToEdit"
      @refresh="refresh"
    />
  </div>
</template>
