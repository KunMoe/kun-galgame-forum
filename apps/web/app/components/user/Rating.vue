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

const { paginationPage, isPaginationPending } = usePaginationScroll(
  toRef(pageData, 'page'),
  () => status.value === 'pending'
)
</script>

<template>
  <div ref="paginationStart" class="pagination-results scroll-mt-20 space-y-3">
    <GalgameRatingCardSkeleton v-if="pending" :count="8" />
    <div v-if="data && data.items.length" class="flex flex-col gap-3">
      <KunLoading :loading="isPaginationPending">
        <GalgameRatingCard :ratings="ratings" :is-transparent="false" />
      </KunLoading>

      <KunPagination
        v-if="data.total > pageData.limit"
        v-model:current-page="paginationPage"
        :total-page="Math.ceil(data.total / pageData.limit)"
        :is-loading="status === 'pending'"
      />
    </div>

    <KunNull v-if="data && !data.items.length" description="暂无评分" />
  </div>
</template>
