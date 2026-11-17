<script setup lang="ts">
const props = defineProps<{ browses: AgentBrowse[] }>()

const links = computed(() =>
  props.browses.flatMap((b) => agentBrowseLink(b) ?? [])
)
</script>

<template>
  <div v-if="links.length" class="flex flex-col gap-2">
    <KunButton
      v-for="l in links"
      :key="l.key"
      :href="l.href"
      variant="flat"
      color="primary"
      size="sm"
      class-name="h-auto min-h-8 w-fit py-1.5 whitespace-normal text-left"
    >
      <KunIcon name="lucide:list-filter" class="size-4 shrink-0" />
      <span>{{ l.label }}</span>
    </KunButton>
  </div>
</template>
