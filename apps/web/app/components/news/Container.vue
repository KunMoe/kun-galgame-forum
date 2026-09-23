<script setup lang="ts">
import { useRouteQuery } from '@vueuse/router'

const opts = { mode: 'replace' as const }
const lane = useRouteQuery<string>('lane', 'all', opts)
const source = useRouteQuery<string>('source', 'all', opts)
const year = useRouteQuery('year', 0, { ...opts, transform: Number })
const month = useRouteQuery('month', 0, { ...opts, transform: Number })

const {
  filters,
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

const { directory: partners } = await useNewsSources()

const { data: archive } = await useApi(
  () =>
    `news-archive:${filters.lane.value ?? ''}:${filters.source.value ?? ''}:${filters.year.value ?? 0}`,
  (api, { signal }) =>
    api.GET('/news-archive', {
      params: {
        query: {
          ...(filters.lane.value
            ? { lane: filters.lane.value as KunNewsItem['lane'] }
            : {}),
          ...(filters.source.value
            ? { news_source: filters.source.value }
            : {}),
          ...(filters.year.value ? { year: filters.year.value } : {})
        }
      },
      signal
    })
)

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
  lane.value = 'all'
  source.value = 'all'
  year.value = 0
  month.value = 0
}
</script>

<template>
  <div class="space-y-6 pb-12">
    <KunHeader
      name="Galgame 情报 / Gal 业界新闻与专栏"
      description="这里聚合了 Galgame 业界的新作情报、发售消息与深度专栏, 全部由合作站点授权转载。本站只做索引, 点击标题或“阅读原文”即可前往对方站点阅读全文。可以按栏目、来源与年份月份筛选, 快速回溯往期情报。"
    />

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
          <KunNull v-if="error" description="情报服务暂时不可用，请稍后再试" />
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

            <div v-if="isLoadingMore" class="space-y-3">
              <KunSkeleton
                v-for="n in 3"
                :key="`skeleton-${n}`"
                height="10rem"
                rounded="lg"
              />
            </div>
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
