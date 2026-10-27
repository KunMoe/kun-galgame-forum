<template>
  <div class="space-y-3">
    <KunCard :is-transparent="false" :is-hoverable="false">
      <div class="flex gap-3">
        <div class="w-20 shrink-0">
          <KunSkeleton height="auto" rounded="lg" class-name="aspect-[3/4]" />
        </div>
        <div class="min-w-0 flex-1 space-y-2">
          <KunSkeleton variant="text" width="55%" class-name="text-lg" />
          <KunSkeleton variant="text" width="35%" class-name="text-sm" />
        </div>
      </div>
    </KunCard>

    <KunCard
      :is-transparent="false"
      :is-hoverable="false"
      content-class="space-y-3"
    >
      <div class="flex items-center gap-3">
        <KunSkeleton variant="circle" width="3.75rem" height="3.75rem" />
        <div class="flex-1 space-y-1">
          <KunSkeleton variant="text" width="8rem" class-name="text-lg" />
          <KunSkeleton variant="text" width="10rem" class-name="text-sm" />
        </div>
        <KunSkeleton height="2.25rem" width="4rem" rounded="lg" />
      </div>
      <div class="flex flex-wrap gap-2">
        <KunSkeleton height="1.75rem" width="5rem" rounded="full" />
        <KunSkeleton height="1.75rem" width="5rem" rounded="full" />
      </div>
      <div>
        <KunSkeleton
          v-for="n in 5"
          :key="n"
          variant="text"
          :width="n === 5 ? '50%' : '100%'"
          class-name="leading-7"
        />
      </div>
    </KunCard>
  </div>
</template>
