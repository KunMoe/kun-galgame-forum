<script setup lang="ts">
withDefaults(
  defineProps<{
    links: { url: string; name: string }[]
    preview?: number
  }>(),
  { preview: 4 }
)

const isExpanded = ref(false)
</script>

<template>
  <div v-if="links.length" class="flex flex-wrap items-center gap-x-4 gap-y-1">
    <template v-for="(link, index) in links" :key="`${index}:${link.url}`">
      <KunLink
        v-if="link.url"
        v-show="isExpanded || index < preview"
        :is-show-anchor-icon="true"
        target="_blank"
        rel="noopener noreferrer"
        underline="hover"
        size="sm"
        :to="link.url"
      >
        {{ link.name }}
      </KunLink>
      <span
        v-else
        v-show="isExpanded || index < preview"
        class="text-default-400 text-sm"
      >
        {{ link.name }}
      </span>
    </template>

    <KunButton
      v-if="links.length > preview"
      variant="light"
      size="sm"
      color="default"
      @click="isExpanded = !isExpanded"
    >
      <KunIcon :name="isExpanded ? 'lucide:chevron-up' : 'lucide:link'" />
      {{ isExpanded ? '收起' : `全部 ${links.length} 个链接` }}
    </KunButton>
  </div>
</template>
