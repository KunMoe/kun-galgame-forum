<template>
  <div class="space-y-4" aria-hidden="true">
    <KunCard
      :is-hoverable="false"
      :is-transparent="false"
      content-class="space-y-6"
    >
      <div class="space-y-3">
        <KunSkeleton variant="text" width="60%" class-name="text-2xl" />
        <div class="flex flex-wrap gap-2">
          <KunSkeleton
            v-for="n in 4"
            :key="n"
            height="1.5rem"
            width="4rem"
            rounded="full"
          />
        </div>
        <div class="pt-6">
          <KunSkeleton
            v-for="n in 6"
            :key="n"
            variant="text"
            :width="n === 6 ? '50%' : '100%'"
            class-name="leading-7"
          />
        </div>
      </div>
      <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
        <KunSkeleton height="12rem" rounded="lg" />
        <div class="space-y-3">
          <KunSkeleton variant="text" width="4rem" />
          <KunSkeleton height="2rem" width="10rem" rounded="full" />
          <KunSkeleton variant="text" width="80%" />
        </div>
      </div>
    </KunCard>
  </div>
</template>
