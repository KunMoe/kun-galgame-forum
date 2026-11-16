<script setup lang="ts">
import type { RatingPage } from '#shared/utils/api/schemas'
import { ratingToCard } from '~/utils/galgame/ratingCard'

const props = defineProps<{
  userId: number
}>()

const { allowsNsfw: includeNsfw } = useContentStance()
const nameOf = useCatalogName()
const pageData = reactive({
  page: usePageQuery(),
  limit: 24
})

const { data, status, pending } = useKunApi<RatingPage>(
  () =>
    `user-ratings:${props.userId}:${pageData.page}:${pageData.limit}:${includeNsfw.value ? 'nsfw' : 'sfw'}`,
  (client, { signal }) =>
    client.GET('/ratings', {
      params: {
        query: {
          author_id: String(props.userId),
          sort: 'created_desc',
          page: pageData.page,
          limit: pageData.limit,
          include_nsfw: includeNsfw.value
        }
      },
      signal
    })
)

const ratings = computed(() =>
  (data.value?.items ?? []).map((r) => ratingToCard(r, nameOf))
)

const listStart = useTemplateRef('listStart')
const revealList = useRevealListStart(
  listStart,
  () => pageData.page,
  () => status.value === 'pending'
)
</script>

<template>
  <div ref="listStart" class="scroll-mt-36 space-y-3 lg:scroll-mt-20">
    <h2 class="sr-only">评分<KunPageLabel :page="pageData.page" /></h2>
    <GalgameRatingCardSkeleton v-if="pending" :count="8" />
    <div v-if="data && data.items.length" class="space-y-3">
      <GalgameRatingCard :ratings="ratings" :is-transparent="false" />

      <KunPagination
        v-if="data.total > pageData.limit"
        v-model:current-page="pageData.page"
        :total-page="pageCountOf(data.total, pageData.limit)"
        :is-loading="status === 'pending'"
        @update:current-page="revealList"
      />
    </div>

    <KunNull v-if="data && !data.items.length" description="暂无评分" />
  </div>
</template>
