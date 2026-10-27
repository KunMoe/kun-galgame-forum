<script setup lang="ts">
const { lane, source, year, month, set } = useQueryState({
  lane: queryString('all'),
  source: queryString('all'),
  year: queryInt(0, 0),
  month: queryInt(0, 0)
})

const { directory: partners } = useNewsSources()

const archiveLane = computed(() =>
  lane.value && lane.value !== 'all' ? lane.value : undefined
)
const archiveSource = computed(() =>
  source.value && source.value !== 'all' ? source.value : undefined
)
const { data: archive, pending: archivePending } = useKunApi(
  () =>
    `news-archive:${archiveLane.value ?? ''}:${archiveSource.value ?? ''}:${year.value > 0 ? year.value : 0}`,
  (api, { signal }) =>
    api.GET('/news-archive', {
      params: {
        query: {
          ...(archiveLane.value
            ? { lane: archiveLane.value as KunNewsItem['lane'] }
            : {}),
          ...(archiveSource.value ? { news_source: archiveSource.value } : {}),
          ...(year.value > 0 ? { year: year.value } : {})
        }
      },
      signal
    })
)

const {
  items,
  groups,
  total,
  status,
  error,
  hasMore,
  isLoadingMore,
  loadMore,
  sentinel
} = await useNewsFeed({ limit: 20, maxAutoLoads: 8, lane, source, year, month })

const isEmpty = computed(
  () => status.value !== 'pending' && !items.value.length
)
const isFiltered = computed(
  () => lane.value !== 'all' || source.value !== 'all' || year.value > 0
)

const range = computed(() => {
  if (!year.value) return ''
  return month.value ? `${year.value} 年 ${month.value} 月` : `${year.value} 年`
})

const reset = () => {
  set({ lane: 'all', source: 'all', year: 0, month: 0 })
}
</script>

<template>
  <div class="space-y-6 pb-12">
    <KunHeader
      name="Galgame 情报 / Gal 业界新闻与专栏"
      description="这里聚合了 Galgame 业界的新作情报、发售消息与深度专栏。合作站点授权转载的情报, 点击标题或“阅读原文”前往对方站点阅读全文; NextMoe 用户投稿的情报可以直接在本站阅读。可以按栏目、来源与年份月份筛选, 快速回溯往期情报。"
    >
      <template #endContent>
        <div class="flex gap-2">
          <KunButton size="sm" href="/edit/news">投稿情报</KunButton>
          <KunButton size="sm" variant="flat" href="/news/mine">
            我的投稿
          </KunButton>
        </div>
      </template>
    </KunHeader>

    <div
      class="grid grid-cols-1 items-start gap-6 lg:grid-cols-[15rem_minmax(0,1fr)]"
    >
      <aside class="min-w-0 lg:sticky lg:top-20 lg:col-start-1 lg:space-y-5">
        <NewsFilters
          v-model:lane="lane"
          v-model:source="source"
          v-model:year="year"
          v-model:month="month"
          :sources="partners"
          :archive="archive"
          :archive-pending="archivePending"
          orientation="horizontal"
          class="lg:hidden"
        />
        <NewsFilters
          v-model:lane="lane"
          v-model:source="source"
          v-model:year="year"
          v-model:month="month"
          :sources="partners"
          :archive="archive"
          :archive-pending="archivePending"
          class="hidden lg:block"
        />
        <NewsPartners :sources="partners" class="hidden lg:block" />
      </aside>

      <div class="min-w-0 space-y-4 lg:col-start-2">
        <div class="flex flex-wrap items-center gap-2">
          <span class="text-default-500 text-sm">
            {{ `共 ${total} 条情报` }}
          </span>
          <KunChip v-if="range" size="sm" color="primary" variant="flat">
            {{ range }}
          </KunChip>
          <KunButton
            v-if="isFiltered"
            size="sm"
            variant="light"
            class-name="ml-auto"
            @click="reset"
          >
            清除筛选
          </KunButton>
        </div>

        <KunLoadingDim :loading="status === 'pending' && items.length > 0">
          <NewsCardSkeletonList
            v-if="status === 'pending' && !items.length"
            :count="5"
            height="10rem"
          />
          <KunNull
            v-else-if="error"
            description="情报服务暂时不可用，请稍后再试"
          />
          <KunNull v-else-if="isEmpty" description="没有符合条件的情报" />

          <div v-else class="space-y-8">
            <section v-for="group in groups" :key="group.key" class="space-y-3">
              <NewsGroupHeader :group="group" />

              <div class="space-y-3">
                <NewsCard
                  v-for="item in group.items"
                  :key="item.id"
                  :item="item"
                  :source="group.source"
                  size="md"
                />
              </div>
            </section>

            <NewsCardSkeletonList v-if="isLoadingMore" height="10rem" />
          </div>

          <div
            v-if="items.length"
            ref="sentinel"
            class="flex justify-center pt-6"
          >
            <KunButton
              v-if="hasMore && !isLoadingMore"
              variant="light"
              @click="loadMore(false)"
            >
              加载更多
            </KunButton>
            <span v-else-if="!hasMore" class="text-default-400 text-sm">
              没有更多情报了
            </span>
          </div>
        </KunLoadingDim>
      </div>
    </div>

    <NewsPartners :sources="partners" class="lg:hidden" />
  </div>
</template>
