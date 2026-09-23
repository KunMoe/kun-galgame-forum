<script setup lang="ts">
import type { WorkRef } from '#shared/utils/api/schemas'

const props = defineProps<{
  work: WorkRef
}>()

const workName = useWorkName()
const name = computed(() => workName(props.work))
</script>

<template>
  <KunCard :is-hoverable="false" :is-transparent="false">
    <div class="grid grid-cols-[7rem_1fr] gap-4 sm:grid-cols-[10rem_1fr]">
      <div class="relative aspect-[3/4] w-full">
        <KunImage
          v-if="work.cover"
          class="size-full rounded-lg object-cover"
          :src="work.cover.url"
          loading="eager"
          fetchpriority="high"
          :thumbhash="work.cover.thumbhash ?? undefined"
          :alt="name"
        />
        <div v-else class="bg-default-100 size-full rounded-lg" />

        <KunChip
          :color="work.is_nsfw ? 'danger' : 'success'"
          class-name="absolute top-2 left-2"
          variant="solid"
        >
          {{ work.is_nsfw ? 'NSFW' : 'SFW' }}
        </KunChip>
      </div>

      <div class="flex min-w-0 flex-col gap-3">
        <div>
          <h2 class="text-2xl font-bold">
            <KunLink
              underline="none"
              color="default"
              :to="`/galgame/${work.id}`"
              class-name="text-2xl hover:text-primary transition-colors"
            >
              {{ name }}
            </KunLink>
          </h2>
        </div>

        <div class="mt-auto flex flex-wrap items-center justify-end gap-2">
          <KunButton variant="flat" href="/galgame"> 浏览更多资源 </KunButton>
          <KunButton :href="`/galgame/${work.id}`">
            查看这个 Galgame 的更多资源
          </KunButton>
        </div>
      </div>
    </div>
  </KunCard>
</template>
