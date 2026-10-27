<template>
  <div class="mx-auto max-w-3xl space-y-4">
    <KunSkeleton variant="text" width="9rem" class-name="text-sm" />
    <KunCard :is-hoverable="false" padding="lg" content-class="gap-6">
      <KunSkeleton height="16rem" rounded="lg" />
      <div class="space-y-3">
        <KunSkeleton height="1.5rem" width="5rem" rounded="full" />
        <KunSkeleton variant="text" width="75%" class-name="text-2xl sm:text-3xl" />
        <KunSkeleton variant="text" width="14rem" class-name="text-sm" />
      </div>
      <div class="space-y-0">
        <KunSkeleton v-for="n in 3" :key="n" variant="text" class-name="leading-7" />
      </div>
      <div>
        <KunSkeleton v-for="n in 6" :key="n" variant="text" :width="n === 6 ? '60%' : '100%'" class-name="leading-7" />
      </div>
    </KunCard>
  </div>
</template>
