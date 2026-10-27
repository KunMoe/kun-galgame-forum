<template>
  <div
    class="flex flex-col gap-8 lg:flex-row lg:items-start"
    aria-hidden="true"
  >
    <KunSkeleton
      width="15rem"
      height="20rem"
      rounded="lg"
      class-name="hidden lg:block"
    />
    <div class="min-w-0 flex-1 space-y-8">
      <GalgameEntityHeaderSkeleton />
      <div class="space-y-3">
        <KunSkeleton variant="text" width="6rem" class-name="text-lg" />
        <div>
          <KunSkeleton v-for="n in 4" :key="n" variant="text" />
          <KunSkeleton variant="text" width="55%" />
        </div>
      </div>
      <div class="flex flex-wrap gap-1.5">
        <KunSkeleton
          v-for="n in 8"
          :key="n"
          height="1.75rem"
          :width="`${4 + (n % 3)}rem`"
          rounded="full"
        />
      </div>
    </div>
  </div>
</template>
