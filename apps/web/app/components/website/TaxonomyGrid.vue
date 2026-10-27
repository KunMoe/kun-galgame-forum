<script setup lang="ts">
const props = defineProps<{
  filter: { website_category_id?: string; website_tag_id?: string }
  emptyText: string
}>()

const { data: websites, pending } = useWebsiteList(() => props.filter)
const sorted = computed(() => [...(websites.value ?? [])].sort(byScore))
</script>

<template>
  <div class="space-y-6">
    <slot name="header" :count="sorted.length" />

    <WebsiteCardSkeleton v-if="pending" />

    <div
      v-else-if="sorted.length"
      class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3"
    >
      <WebsiteCard
        v-for="website in sorted"
        :key="website.id"
        :website="website"
      />
    </div>

    <KunNull v-else :description="emptyText" />
  </div>
</template>
