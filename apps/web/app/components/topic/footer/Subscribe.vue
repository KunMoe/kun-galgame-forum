<script setup lang="ts">
import {
  TOPIC_NOTIFICATION_OPTIONS,
  useTopicSubscription
} from '~/composables/topic/useTopicSubscription'

const props = defineProps<{
  topicId: string
}>()

const { subscription, busy, load, setLevel } = useTopicSubscription(
  () => props.topicId
)

const current = computed(
  () =>
    TOPIC_NOTIFICATION_OPTIONS.find(
      (option) =>
        option.value === (subscription.value?.notification_level ?? 'normal')
    ) ?? TOPIC_NOTIFICATION_OPTIONS[1]!
)

onMounted(() => {
  void load()
})
</script>

<template>
  <KunPopover position="top-start" inner-class="w-72 p-2">
    <template #trigger>
      <KunTooltip :text="`话题通知: ${current.label}`">
        <KunReaction
          :toggle="false"
          :icon="current.icon"
          :label="`话题通知: ${current.label}`"
        />
      </KunTooltip>
    </template>

    <div class="flex flex-col gap-1">
      <p class="text-default-500 px-2 pt-1 pb-2 text-xs">
        选择这个话题有新回复时如何通知你
      </p>
      <KunButton
        v-for="option in TOPIC_NOTIFICATION_OPTIONS"
        :key="option.value"
        variant="light"
        :color="option.value === current.value ? 'primary' : 'default'"
        size="sm"
        :disabled="busy"
        class-name="h-auto w-full justify-start gap-3 py-2 text-left"
        @click="setLevel(option.value)"
      >
        <KunIcon class-name="shrink-0 text-lg" :name="option.icon" />
        <span class="flex min-w-0 flex-1 flex-col">
          <span class="font-medium">{{ option.label }}</span>
          <span class="text-default-500 text-xs whitespace-normal">
            {{ option.description }}
          </span>
        </span>
        <KunIcon
          v-if="option.value === current.value"
          class-name="shrink-0"
          name="lucide:check"
        />
      </KunButton>
    </div>
  </KunPopover>
</template>
