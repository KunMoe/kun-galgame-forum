<script setup lang="ts">
import type { ListCollectionSummary } from '#shared/utils/api/schemas'

const props = defineProps<{
  workId: number
}>()

const LIMIT = 6
const { allowsNsfw, stanceKey } = useContentStance()

const { data } = useKunApi<ListCollectionSummary>(
  () => `collections-containing:${props.workId}:${stanceKey.value}`,
  (api, { signal }) =>
    api.GET('/collections', {
      params: {
        query: {
          contains_work_id: String(props.workId),
          include_nsfw: allowsNsfw.value,
          limit: LIMIT
        }
      },
      signal
    }),
  { server: false }
)
</script>

<template>
  <div v-if="data?.items.length" class="space-y-3">
    <KunHeader
      name="收录了这部作品的收藏夹"
      description="其他玩家整理的游戏清单"
      scale="h3"
    >
      <template v-if="data.next_cursor" #endContent>
        <KunLink :to="`/collection?work=${workId}`" size="sm" underline="hover">
          查看全部
        </KunLink>
      </template>
    </KunHeader>
    <div class="grid grid-cols-1 gap-2 md:grid-cols-2">
      <GalgameCollectionRowCard
        v-for="c in data.items"
        :key="c.id"
        :collection="c"
      />
    </div>
  </div>
</template>
