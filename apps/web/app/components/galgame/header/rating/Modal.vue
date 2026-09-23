<script setup lang="ts">
import type { KunTabItem } from '@kungal/ui-vue'
import {
  KUN_GALGAME_EXTERNAL_RATING_CONST,
  KUN_GALGAME_EXTERNAL_RATING_MAP,
  KUN_GALGAME_LOCAL_RATING_META,
  KUN_GALGAME_LOCAL_RATING_SOURCE
} from '~/constants/galgame-rating'
import type { Work } from '#shared/utils/api/schemas'
import type { GalgameRatingCardOnGalgamePage } from '~~/shared/types/galgame-rating'

const props = defineProps<{
  galgame: Work
  ratings: GalgameRatingCardOnGalgamePage[]
  source: string
}>()

const open = defineModel<boolean>({ required: true })
const workName = useWorkName()

const externalSources = computed(() =>
  KUN_GALGAME_EXTERNAL_RATING_CONST.filter((source) =>
    props.galgame.external_ratings.some((row) => row.site === source)
  )
)

const tabs = computed<KunTabItem[]>(() => [
  ...(props.ratings.length
    ? [
        {
          value: KUN_GALGAME_LOCAL_RATING_SOURCE,
          textValue: KUN_GALGAME_LOCAL_RATING_META.label,
          icon: 'lucide:star'
        }
      ]
    : []),
  ...externalSources.value.map((source) => ({
    value: source,
    textValue: KUN_GALGAME_EXTERNAL_RATING_MAP[source].label
  }))
])

const active = ref(props.source)

watch([open, () => props.source], ([isOpen, next]) => {
  if (!isOpen) return
  active.value = tabs.value.some((tab) => tab.value === next)
    ? next
    : (tabs.value[0]?.value ?? KUN_GALGAME_LOCAL_RATING_SOURCE)
})
</script>

<template>
  <KunModal
    v-model="open"
    inner-class-name="max-w-3xl w-full"
    scroll-behavior="inside"
    aria-label="评分详情"
  >
    <div class="space-y-4">
      <div>
        <h3 class="text-lg font-bold">评分详情</h3>
        <p class="text-default-500 line-clamp-1 text-sm">
          {{ workName(galgame) }}
        </p>
      </div>

      <KunTab
        v-if="tabs.length > 1"
        v-model="active"
        :items="tabs"
        variant="light"
        size="sm"
      />

      <KunTabPanels v-model="active" mount="lazy">
        <KunTabPanel
          v-if="ratings.length"
          :value="KUN_GALGAME_LOCAL_RATING_SOURCE"
        >
          <GalgameHeaderRatingLocalPanel
            :galgame="galgame"
            :ratings="ratings"
          />
        </KunTabPanel>

        <KunTabPanel
          v-for="external in externalSources"
          :key="external"
          :value="external"
        >
          <GalgameHeaderRatingExternalPanel
            :galgame="galgame"
            :source="external"
          />
        </KunTabPanel>
      </KunTabPanels>
    </div>
  </KunModal>
</template>
