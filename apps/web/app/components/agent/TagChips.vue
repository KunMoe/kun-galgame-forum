<script setup lang="ts">
defineProps<{ tags: { id: string; name: string; href: string }[] }>()
</script>

<template>
  <div class="flex flex-wrap gap-1.5">
    <NuxtLink v-for="t in tags" :key="t.id" :to="t.href">
      <KunChip size="sm" variant="flat" color="warning">
        {{ t.name }}
      </KunChip>
    </NuxtLink>
  </div>
</template>
