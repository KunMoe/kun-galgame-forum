<script setup lang="ts">
defineProps<{ title: string; note: string }>()
</script>

<template>
  <section class="space-y-5">
    <div class="space-y-1">
      <h2 class="text-foreground text-xl font-semibold">{{ title }}</h2>
      <p class="text-default-500 text-sm">{{ note }}</p>
    </div>
    <div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
      <slot />
    </div>
  </section>
</template>
