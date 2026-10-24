<script setup lang="ts">
import { ref } from 'vue'
import type { Poll } from '#shared/utils/api/schemas'
import type { PollFormData } from './types'

const props = defineProps<{
  topicId: number
  isTopicAdmin: boolean
  createForm?: PollFormData
}>()

const emits = defineEmits<{
  created: []
}>()

const isCreateOpen = defineModel<boolean>('isCreateOpen', { default: false })

const topicId = computed(() => String(props.topicId))
const isModalOpen = ref(false)
const pollToEdit = ref<Poll | undefined>(undefined)
const formToCreate = ref<PollFormData | undefined>(undefined)

const { data, refresh } = await useApi(
  () => `topic-polls:${topicId.value}`,
  (api, { signal }) =>
    api.GET('/topics/{topic_id}/polls', {
      params: { path: { topic_id: topicId.value } },
      signal
    })
)

const polls = computed(() => data.value?.items ?? [])

watch(isCreateOpen, (open) => {
  if (!open) {
    return
  }
  pollToEdit.value = undefined
  formToCreate.value = props.createForm
  isModalOpen.value = true
  isCreateOpen.value = false
})

const openEditModal = (poll: Poll) => {
  pollToEdit.value = poll
  formToCreate.value = undefined
  isModalOpen.value = true
}

const onSaved = () => {
  if (!pollToEdit.value) {
    emits('created')
  }
  refresh()
}
</script>

<template>
  <div class="space-y-3">
    <TopicPollList
      v-for="poll in polls"
      :key="poll.id"
      :poll="poll"
      @edit="openEditModal"
      @refresh="refresh"
    />

    <TopicPollModal
      v-if="isTopicAdmin"
      v-model="isModalOpen"
      :topic-id="topicId"
      :initial-data="pollToEdit"
      :initial-form="formToCreate"
      @refresh="onSaved"
    />
  </div>
</template>
