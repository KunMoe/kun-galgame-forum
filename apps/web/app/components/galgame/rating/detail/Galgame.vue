<script setup lang="ts">
import type { WorkSummary } from '#shared/utils/api/schemas'

const props = defineProps<{
  work: WorkSummary
}>()

const nameOf = useCatalogName()
const name = computed(() => nameOf(props.work).name)
const maker = computed(() =>
  props.work.maker
    ? { id: props.work.maker.id, name: nameOf(props.work.maker).name }
    : null
)
</script>

<template>
  <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
    <div
      class="relative aspect-video h-full w-full overflow-hidden rounded-lg md:col-span-1"
    >
      <KunImage
        class="size-full rounded-lg object-cover"
        :src="work.banner?.url ?? ''"
        loading="lazy"
        :thumbhash="work.banner?.thumbhash ?? ''"
        :alt="name"
      />
    </div>

    <div class="space-y-3">
      <KunLink :to="`/galgame/${work.id}`" underline="none">
        <h1
          class="text-content hover:text-primary text-lg font-bold transition-colors sm:text-2xl"
        >
          {{ name }}
        </h1>
      </KunLink>

      <div class="text-default-500 flex items-center gap-3">
        <div class="flex items-center gap-2">
          <KunIcon class-name="text-warning text-2xl" name="lucide:lollipop" />
          <span class="text-warning text-xl font-bold">
            {{ work.rating_score?.toFixed(1) ?? '0.0' }}
          </span>
        </div>
        <span class="bg-default-300 h-3 w-px" />
        <div class="flex items-center gap-2">
          <KunIcon name="lucide:users" />
          <span>{{ work.rating_count }} 人评分</span>
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <template v-if="maker">
          <span
            class="text-default-500 dark:text-default-400 text-sm font-medium"
          >
            制作会社
          </span>
          <KunLink :to="`/galgame/official/${maker.id}`" size="sm">
            {{ maker.name }}
          </KunLink>
        </template>

        <span v-if="maker && work.release_date" class="bg-default-300 h-3 w-px" />

        <template v-if="work.release_date">
          <span
            class="text-default-500 dark:text-default-400 text-sm font-medium"
          >
            发售日期
          </span>
          <KunChip color="warning" variant="flat">
            {{ work.release_date }}
          </KunChip>
        </template>
      </div>
    </div>
  </div>
</template>
