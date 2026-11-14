<script setup lang="ts">
import type { PageListCollectionSummary } from '#shared/utils/api/schemas'

const props = defineProps<{
  userId: number
  ownerName: string | null
}>()

const { allowsNsfw, stanceKey } = useContentStance()
const page = usePageQuery()
const limit = 24

const { data, status, pending } = useKunApi<PageListCollectionSummary>(
  () =>
    `user-collections:${props.userId}:${page.value}:${limit}:${stanceKey.value}`,
  (api, { signal }) =>
    api.GET('/users/{user_id}/collections', {
      params: {
        path: { user_id: String(props.userId) },
        query: {
          page: page.value,
          limit,
          include_nsfw: allowsNsfw.value
        }
      },
      signal
    })
)

const listStart = useTemplateRef('listStart')
const revealList = useRevealListStart(
  listStart,
  page,
  () => status.value === 'pending'
)
</script>

<template>
  <div ref="listStart" class="scroll-mt-36 space-y-3 lg:scroll-mt-20">
    <h2 class="sr-only">游戏收藏夹<KunPageLabel :page="page" /></h2>
    <GalgameCollectionRowCardSkeleton v-if="pending" :count="4" />
    <div v-if="data && data.items.length" class="flex flex-col space-y-3">
      <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
        <GalgameCollectionRowCard
          v-for="c in data.items"
          :key="c.id"
          :collection="c"
          :show-owner="false"
        />
      </div>

      <KunPagination
        v-if="data.total > limit"
        v-model:current-page="page"
        :total-page="Math.ceil(data.total / limit)"
        :is-loading="status === 'pending'"
        @update:current-page="revealList"
      />
    </div>

    <KunNull
      v-if="data && !data.items.length"
      description="这只笨蛋萝莉还没有任何收藏夹"
    />
  </div>
</template>
