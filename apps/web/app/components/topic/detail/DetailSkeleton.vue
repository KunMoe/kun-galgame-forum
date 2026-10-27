<template>
  <div class="flex flex-col gap-4 lg:flex-row lg:items-start">
    <div class="hidden w-52 shrink-0 flex-col items-center gap-3 pt-3 lg:flex">
      <KunSkeleton variant="circle" width="5.5rem" height="5.5rem" />
      <KunSkeleton variant="text" width="7rem" class-name="text-lg" />
      <KunSkeleton variant="text" width="5rem" class-name="text-sm" />
    </div>

    <div class="min-w-0 flex-1 space-y-4">
      <KunCard :is-hoverable="false" content-class="gap-4 justify-start">
        <div class="space-y-3">
          <KunSkeleton
            variant="text"
            width="70%"
            class-name="text-3xl leading-tight lg:text-4xl"
          />
          <div class="flex gap-2">
            <KunSkeleton height="1.5rem" width="4rem" rounded="full" />
            <KunSkeleton height="1.5rem" width="4rem" rounded="full" />
          </div>
          <KunSkeleton variant="text" width="16rem" class-name="text-sm" />
        </div>
        <div>
          <KunSkeleton
            v-for="n in 8"
            :key="n"
            variant="text"
            :width="n === 8 ? '45%' : '100%'"
            class-name="leading-7"
          />
        </div>
      </KunCard>

      <ActivityCardSkeleton v-for="n in 3" :key="n" />
    </div>
  </div>
</template>
