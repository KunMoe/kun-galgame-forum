<script setup lang="ts">
import type { ListWalkthrough } from '#shared/utils/api/schemas'
import { KUN_GALGAME_WALKTHROUGH_SOURCE } from '~/constants/galgameWalkthrough'

const props = defineProps<{
  characterId: string
}>()

const { allowsNsfw, stanceKey } = useContentStance()

const { data } = useKunApi<ListWalkthrough>(
  () => `character-walkthroughs:${props.characterId}:${stanceKey.value}`,
  (api, { signal }) =>
    api.GET('/walkthroughs', {
      params: {
        query: {
          character_id: props.characterId,
          include_nsfw: allowsNsfw.value
        }
      },
      signal
    }),
  { lazy: true, server: false }
)

const items = computed(() =>
  (data.value?.items ?? []).map((w) => ({
    walkthrough: w,
    key: w.chart_characters.find(
      (c) => c.catalog_character?.id === props.characterId
    )?.key
  }))
)
</script>

<template>
  <section v-if="items.length" class="space-y-3">
    <h2 class="text-foreground text-lg font-semibold">攻略</h2>
    <div class="flex flex-wrap gap-2">
      <KunButton
        v-for="{ walkthrough, key } in items"
        :key="walkthrough.id"
        :href="`/galgame/${walkthrough.work_id}/walkthrough/${walkthrough.id}${key ? `?character=${key}` : ''}`"
        variant="flat"
        color="primary"
        size="sm"
      >
        <KunIcon name="lucide:route" />
        查看这名角色的攻略路线
        <span class="text-xs opacity-70">
          ({{
            KUN_GALGAME_WALKTHROUGH_SOURCE[walkthrough.source_site] ??
            walkthrough.source_site
          }})
        </span>
      </KunButton>
    </div>
  </section>
</template>
