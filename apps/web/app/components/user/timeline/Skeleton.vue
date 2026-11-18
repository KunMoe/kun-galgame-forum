<script setup lang="ts">
withDefaults(defineProps<{ days?: number; entries?: number }>(), {
  days: 2,
  entries: 2
})
</script>

<template>
  <div class="space-y-6" aria-hidden="true">
    <section v-for="day in days" :key="day" class="space-y-3">
      <KunSkeleton variant="text" width="4.5rem" class-name="text-sm" />
      <div
        class="before:bg-default-200 relative space-y-6 before:absolute before:top-4 before:bottom-4 before:left-4 before:w-px"
      >
        <div v-for="n in entries" :key="n" class="relative pl-11">
          <KunSkeleton
            variant="circle"
            width="2rem"
            height="2rem"
            class-name="absolute top-0 left-0"
          />
          <div class="flex min-h-8 items-center gap-2">
            <KunSkeleton
              variant="text"
              :width="n % 2 ? '9rem' : '7rem'"
              class-name="text-sm"
            />
            <KunSkeleton
              variant="text"
              width="3rem"
              class-name="ml-auto text-xs"
            />
          </div>
          <div class="mt-2 flex gap-3">
            <KunSkeleton
              width="5rem"
              height="auto"
              rounded="md"
              class-name="aspect-[3/4] shrink-0"
            />
            <div class="min-w-0 flex-1 space-y-2">
              <KunSkeleton variant="text" width="60%" class-name="text-sm" />
              <KunSkeleton variant="text" width="40%" class-name="text-xs" />
              <div class="flex gap-1">
                <KunSkeleton
                  v-for="tag in 3"
                  :key="tag"
                  height="1.25rem"
                  width="3rem"
                  rounded="md"
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>
