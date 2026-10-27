<script setup lang="ts">
import { ref } from 'vue'
import type { Lottery } from '#shared/utils/api/schemas'
import type { LotteryFormData } from './types'

const props = defineProps<{
  topicId: number
  isTopicAdmin: boolean
  createForm?: LotteryFormData
}>()

const emits = defineEmits<{
  created: []
}>()

const isCreateOpen = defineModel<boolean>('isCreateOpen', { default: false })

const topicId = computed(() => String(props.topicId))
const { allowsNsfw, stanceKey } = useContentStance()
const isModalOpen = ref(false)
const lotteryToEdit = ref<Lottery | undefined>(undefined)
const formToCreate = ref<LotteryFormData | undefined>(undefined)

const { data, refresh } = useKunApi(
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
  formToCreate.value = props.createForm
  isModalOpen.value = true
  isCreateOpen.value = false
})

const openEditModal = (lottery: Lottery) => {
  lotteryToEdit.value = lottery
  formToCreate.value = undefined
  isModalOpen.value = true
}

const onSaved = () => {
  if (!lotteryToEdit.value) {
    emits('created')
  }
  refresh()
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
      :initial-form="formToCreate"
      @refresh="onSaved"
    />
  </div>
</template>
