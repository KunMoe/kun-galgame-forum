<script setup lang="ts">
import {
  KUN_GALGAME_OFFICIAL_CATEGORY_MAP,
  KUN_GALGAME_OFFICIAL_LANGUAGE_MAP
} from '~/constants/galgameOfficial'

const props = defineProps<{
  official: GalgameOfficialItem
}>()

const detail = computed(() => props.official)

const categoryText = (category: string) =>
  KUN_GALGAME_OFFICIAL_CATEGORY_MAP[category] || category

const logoSrc = computed(() =>
  props.official.logo ? withImageVariant(props.official.logo, 'mini') : ''
)
</script>

<template>
  <KunCard
    :is-transparent="false"
    :is-hoverable="true"
    :href="`/galgame/official/${official.id}`"
  >
    <div class="flex items-center gap-2">
      <KunImage
        v-if="logoSrc"
        :src="logoSrc"
        :alt="`${official.name} logo`"
        loading="lazy"
        object-fit="contain"
        class-name="size-10 shrink-0 rounded-md"
      />
      <h3 class="text-default-900 min-w-0 font-semibold">
        {{ official.name }}
        <KunChip v-if="detail" size="xs">
          {{ `+ ${detail.galgame_count}` }}
        </KunChip>
      </h3>
    </div>
    <div v-if="detail" class="flex items-center gap-x-2">
      <KunChip size="xs" class-name="rounded-md" color="primary">
        {{ categoryText(detail.category) }}
      </KunChip>
      <span
        v-if="detail.lang"
        class="text-default-500 dark:text-default-400 text-xs"
      >
        {{ KUN_GALGAME_OFFICIAL_LANGUAGE_MAP[detail.lang] || detail.lang }}
      </span>
    </div>
    <div
      v-if="detail?.alias.length"
      class="text-default-500 flex flex-wrap gap-2"
    >
      <KunChip
        size="xs"
        color="success"
        v-for="(a, index) in detail.alias"
        :key="index"
      >
        {{ a }}
      </KunChip>
    </div>
  </KunCard>
</template>
