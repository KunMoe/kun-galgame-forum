<template>
  <div class="flex flex-col gap-3">
    <KunCard
      :is-hoverable="false"
      :is-transparent="false"
      content-class="grid grid-cols-[7rem_1fr] items-start gap-3 md:grid-cols-[220px_1fr]"
    >
      <KunSkeleton height="auto" class-name="aspect-[5/7]" rounded="lg" />
      <div class="space-y-3">
        <KunSkeleton
          variant="text"
          width="70%"
          class-name="text-2xl font-bold"
        />
        <KunSkeleton variant="text" width="45%" class-name="text-sm" />
        <div class="flex flex-wrap gap-2">
          <KunSkeleton
            v-for="n in 4"
            :key="n"
            height="2rem"
            width="5.5rem"
            rounded="lg"
          />
        </div>
      </div>
    </KunCard>

    <div class="grid grid-cols-1 gap-3 md:grid-cols-3">
      <div
        class="order-1 flex min-w-0 flex-col gap-3 md:col-span-2 md:col-start-2 md:row-start-1"
      >
        <KunSkeleton height="2.5rem" rounded="lg" />
        <KunCard :is-hoverable="false" :is-transparent="false">
          <div class="space-y-3">
            <KunSkeleton
              v-for="n in 6"
              :key="n"
              variant="text"
              :width="n === 6 ? '55%' : '100%'"
              class-name="leading-7"
            />
          </div>
        </KunCard>
      </div>

      <div
        class="order-2 flex min-w-0 flex-col gap-3 md:col-span-1 md:col-start-1 md:row-start-1"
      >
        <KunSkeleton height="9rem" rounded="lg" />
        <KunSkeleton height="12rem" rounded="lg" />
      </div>
    </div>
  </div>
</template>
