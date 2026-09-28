<script setup lang="ts">
defineProps<{ title: string; note: string; icon: string; tint: string }>()
</script>

<template>
  <section class="space-y-4">
    <div class="flex items-center gap-3">
      <div
        :class="
          cn(
            'flex size-9 shrink-0 items-center justify-center rounded-xl',
            tint
          )
        "
      >
        <KunIcon :name="icon" class="size-5" />
      </div>
      <div class="min-w-0">
        <h2 class="text-foreground leading-tight font-semibold">
          {{ title }}
        </h2>
        <p class="text-default-500 mt-0.5 text-xs">{{ note }}</p>
      </div>
    </div>
    <div class="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
      <slot />
    </div>
  </section>
</template>
