<template>
  <div class="space-y-3">
    <KunCard :is-hoverable="false" :is-transparent="false">
      <div class="grid grid-cols-[7rem_1fr] gap-4 sm:grid-cols-[10rem_1fr]">
        <KunSkeleton height="auto" rounded="lg" class-name="aspect-[3/4]" />
        <div class="flex min-w-0 flex-col gap-3">
          <KunSkeleton variant="text" width="65%" class-name="text-2xl" />
          <div class="mt-auto flex justify-end">
            <KunSkeleton height="2.5rem" width="8rem" rounded="lg" />
          </div>
        </div>
      </div>
    </KunCard>

    <div class="grid grid-cols-1 gap-3 lg:grid-cols-3">
      <KunCard
        :is-hoverable="false"
        :is-transparent="false"
        class-name="min-w-0 lg:col-span-2"
        content-class="space-y-4"
      >
        <KunSkeleton variant="text" width="50%" class-name="text-2xl" />
        <KunSkeleton variant="text" class-name="text-sm" />
        <div class="flex flex-wrap gap-2">
          <KunSkeleton
            v-for="n in 3"
            :key="n"
            height="1.75rem"
            width="5rem"
            rounded="full"
          />
        </div>
        <KunSkeleton height="6rem" rounded="lg" />
      </KunCard>

      <div class="min-w-0 space-y-3">
        <GalgameResourceCardSkeleton v-for="n in 3" :key="n" />
      </div>
    </div>
  </div>
</template>
