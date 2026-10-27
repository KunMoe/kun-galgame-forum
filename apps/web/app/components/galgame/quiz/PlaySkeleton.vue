<template>
  <div class="space-y-4">
    <KunSkeleton height="2rem" width="6rem" rounded="lg" />
    <KunCard :is-transparent="false" :is-hoverable="false">
      <div class="space-y-4">
        <div class="flex flex-wrap gap-2">
          <KunSkeleton height="1.75rem" width="5rem" rounded="full" />
          <KunSkeleton height="1.75rem" width="4rem" rounded="full" />
          <KunSkeleton height="1.75rem" width="4.5rem" rounded="full" />
        </div>
        <div>
          <KunSkeleton variant="text" class-name="text-lg leading-7" />
          <KunSkeleton
            variant="text"
            width="65%"
            class-name="text-lg leading-7"
          />
        </div>
        <div class="space-y-2">
          <KunSkeleton v-for="n in 4" :key="n" height="3rem" rounded="lg" />
        </div>
      </div>
    </KunCard>
  </div>
</template>
