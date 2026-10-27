<script setup lang="ts">
withDefaults(defineProps<{ count?: number }>(), { count: 12 })

const { showKUNGalgamePhoneCardColumns } = storeToRefs(
  usePersistSettingsStore()
)
</script>

<template>
  <div class="@container" aria-hidden="true">
    <div
      :class="
        cn(
          'grid grid-cols-2 gap-2 @lg:grid-cols-3 @lg:gap-3 @2xl:grid-cols-4 @4xl:grid-cols-5 @5xl:grid-cols-6',
          showKUNGalgamePhoneCardColumns === 3 &&
            'max-sm:grid-cols-3 max-sm:gap-1.5'
        )
      "
    >
      <div v-for="n in count" :key="n" class="space-y-2">
        <KunSkeleton height="auto" rounded="lg" class-name="aspect-[5/7]" />
        <KunSkeleton variant="text" width="80%" class-name="text-sm" />
      </div>
    </div>
  </div>
</template>
