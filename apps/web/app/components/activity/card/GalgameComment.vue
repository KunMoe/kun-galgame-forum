<script setup lang="ts">
const props = defineProps<{ activity: ActivityItem }>()

const data = computed(
  () => props.activity.data as GalgameActivityData | undefined
)
const workId = computed(() => data.value?.galgame_id ?? 0)
const detailLink = computed(() =>
  workId.value ? `/galgame/${workId.value}` : props.activity.link
)
</script>

<template>
  <ActivityCardShell :actor="activity.actor" :timestamp="activity.timestamp">
    <div class="space-y-1.5">
      <ActivityCardQuote
        v-if="data?.parent_comment"
        :content="data.parent_comment.content"
      />

      <KunContent
        compact
        class="text-base"
        :content="renderKatex(activity.content)"
      />
      <KunLink
        underline="none"
        color="default"
        :to="detailLink"
        class-name="text-default-500 hover:text-primary inline-flex items-center gap-1 text-sm"
      >
        <KunIcon name="lucide:gamepad-2" class="size-3.5 shrink-0" />
        {{ data?.name }}
      </KunLink>
    </div>
  </ActivityCardShell>
</template>
