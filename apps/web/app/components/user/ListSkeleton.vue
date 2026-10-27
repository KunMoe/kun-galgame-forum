<script setup lang="ts">
withDefaults(
  defineProps<{ rows?: number; shape?: 'text' | 'media' | 'user' }>(),
  { rows: 5, shape: 'text' }
)
</script>

<template>
  <div
    :class="cn('flex flex-col', shape === 'user' ? 'space-y-2' : 'space-y-3')"
    aria-hidden="true"
  >
    <KunCard
      v-for="n in rows"
      :key="n"
      :padding="shape === 'user' ? 'sm' : undefined"
      :is-hoverable="false"
      :is-transparent="false"
      :content-class="shape === 'text' ? undefined : 'flex items-center gap-3'"
    >
      <template v-if="shape === 'user'">
        <KunSkeleton variant="circle" width="1.75rem" height="1.75rem" />
        <KunSkeleton variant="text" width="6rem" class-name="text-sm" />
        <KunSkeleton variant="text" width="4rem" class-name="ml-auto text-sm" />
      </template>
      <template v-else>
        <KunSkeleton
          v-if="shape === 'media'"
          width="6rem"
          height="3.5rem"
          rounded="lg"
        />
        <div class="min-w-0 grow">
          <KunSkeleton variant="text" :width="n % 2 ? '60%' : '45%'" />
          <KunSkeleton variant="text" width="7rem" class-name="text-sm" />
        </div>
      </template>
    </KunCard>
  </div>
</template>
