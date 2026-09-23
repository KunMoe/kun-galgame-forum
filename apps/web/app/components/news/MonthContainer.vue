<script setup lang="ts">
import { useRouteQuery } from '@vueuse/router'

const props = defineProps<{ year: number; month: number }>()

const route = useRoute()
const opts = { mode: 'replace' as const }
const day = useRouteQuery('day', 0, { ...opts, transform: Number })
const page = useRouteQuery('page', 1, { ...opts, transform: Number })

// The lane and source a reader arrived with are carried, not edited: the month
// page is one slice of the overview's list, and its counts have to keep
// describing that slice. Changing either is what 返回情报总览 is for.
const lane = computed(() => String(route.query.lane ?? ''))
const source = computed(() => String(route.query.source ?? ''))

const label = `${props.year} 年 ${props.month} 月`

const PAGE_SIZE = 50

const filterQuery = computed(() => ({
  ...(lane.value ? { lane: lane.value as KunNewsItem['lane'] } : {}),
  ...(source.value ? { news_source: source.value } : {})
}))

const sourcesReady = useNewsSources()
const { byKey: sources } = sourcesReady

const monthPath = { year: props.year, month: props.month }

const summaryReady = useApi(
  () => `news-month:${props.year}-${props.month}:${lane.value}:${source.value}`,
  (api, { signal }) =>
    api.GET('/news-archive/{year}/{month}', {
      params: { path: monthPath, query: filterQuery.value },
      signal
    })
)
const itemsReady = useApi(
  () =>
    `news-month-items:${props.year}-${props.month}:${lane.value}:${source.value}:${day.value}:${page.value}`,
  (api, { signal }) =>
    api.GET('/news-archive/{year}/{month}/items', {
      params: {
        path: monthPath,
        query: {
          ...filterQuery.value,
          page: page.value,
          limit: PAGE_SIZE,
          ...(day.value ? { day: day.value } : {})
        }
      },
      signal
    })
)
const archiveReady = useApi(
  () => `news-archive:${lane.value}:${source.value}:${props.year}`,
  (api, { signal }) =>
    api.GET('/news-archive', {
      params: { query: { ...filterQuery.value, year: props.year } },
      signal
    })
)
const [{ data: summary }, { data, status, problem: error }, { data: archive }] =
  await Promise.all([summaryReady, itemsReady, archiveReady, sourcesReady])

const groups = computed(() =>
  groupNewsItems(data.value?.items ?? [], sources.value)
)
const partners = computed(() => {
  const keys = new Set(
    (data.value?.items ?? []).map((item) => item.news_source)
  )
  return [...keys].flatMap((key) =>
    sources.value[key] ? [sources.value[key]] : []
  )
})

// Only the days that published anything. Both partners post weekly digests, so
// a full 31-cell strip would be mostly dead cells.
const days = computed(() => (summary.value?.days ?? []).filter((d) => d.count))
const months = computed(() => archive.value?.months ?? [])
const totalPage = computed(() =>
  Math.ceil((data.value?.total ?? 0) / PAGE_SIZE)
)

const carried = () => {
  const query = new URLSearchParams()
  if (lane.value) query.set('lane', lane.value)
  if (source.value) query.set('source', source.value)
  return query
}

const withSearch = (month: number, query: URLSearchParams) => {
  const search = query.toString()
  return `/news/${props.year}/${month}${search ? `?${search}` : ''}`
}

// Neither an empty month nor the month already open gets an href: rendering
// them as links hands a crawler a dozen dead ends per page, and disabled only
// stops the pointer.
const monthHref = (entry: KunNewsArchiveMonth) =>
  entry.count && entry.month !== props.month
    ? withSearch(entry.month, carried())
    : ''

const pageHref = (n: number) => {
  const query = carried()
  if (day.value) query.set('day', String(day.value))
  if (n > 1) query.set('page', String(n))
  return withSearch(props.month, query)
}

const overviewHref = computed(() => {
  const query = carried()
  query.set('year', String(props.year))
  query.set('month', String(props.month))
  return `/news?${query.toString()}`
})

const selectDay = (value: number) => {
  page.value = 1
  day.value = day.value === value ? 0 : value
}

// Nuxt's scroll behaviour deliberately stays put when only the query changes,
// which is right for a filter and wrong for a paginator: turning to page two
// from the bottom of page one left the reader in the middle of the new page.
watch(page, () => window.scrollTo({ top: 0, behavior: 'smooth' }))
</script>

<template>
  <div class="space-y-6 pb-12">
    <KunHeader
      :name="`${label} Galgame 业界新闻`"
      :description="`${label}的 Galgame 业界新作情报、发售消息与深度专栏存档, 全部由合作站点授权转载。本站只做索引, 点击标题或“阅读原文”即可前往对方站点阅读全文。`"
    >
      <template #endContent>
        <KunButton variant="light" size="sm" :href="overviewHref">
          返回情报总览
        </KunButton>
      </template>
    </KunHeader>

    <KunNull v-if="error" description="情报服务暂时不可用，请稍后再试" />

    <div v-else class="space-y-4">
      <section v-if="months.length" class="space-y-2">
        <h2 class="text-default-500 px-1 text-xs font-medium">
          {{ `${year} 年` }}
        </h2>
        <div class="grid grid-cols-4 gap-1 sm:grid-cols-6 lg:grid-cols-12">
          <KunButton
            v-for="entry in months"
            :key="entry.month"
            size="sm"
            :variant="entry.month === month ? 'flat' : 'light'"
            :disabled="!entry.count"
            :href="monthHref(entry)"
          >
            {{ `${entry.month} 月` }}
            <span class="text-default-400 ml-1 text-xs tabular-nums">
              {{ entry.count }}
            </span>
          </KunButton>
        </div>
      </section>

      <section v-if="days.length" class="space-y-2">
        <h2 class="text-default-500 px-1 text-xs font-medium">按发布日期</h2>
        <div class="flex flex-wrap gap-1.5">
          <KunButton
            size="sm"
            :variant="day ? 'light' : 'flat'"
            @click="selectDay(0)"
          >
            {{ `整月 ${summary?.item_count ?? 0}` }}
          </KunButton>
          <KunButton
            v-for="entry in days"
            :key="entry.day"
            size="sm"
            :variant="day === entry.day ? 'flat' : 'light'"
            @click="selectDay(entry.day)"
          >
            {{ `${entry.day} 日` }}
            <span class="text-default-400 ml-1 text-xs tabular-nums">
              {{ entry.count }}
            </span>
          </KunButton>
        </div>
      </section>

      <div class="flex flex-wrap items-center gap-2">
        <span class="text-default-500 text-sm">
          {{ `共 ${data?.total ?? 0} 条情报` }}
        </span>
        <KunChip v-if="day" size="sm" color="primary" variant="flat">
          {{ `${month} 月 ${day} 日` }}
        </KunChip>
        <span v-if="totalPage > 1" class="text-default-400 text-sm">
          {{ `第 ${page} / ${totalPage} 页` }}
        </span>
      </div>

      <KunLoadingDim :loading="status === 'pending'">
        <KunNull
          v-if="!groups.length"
          :description="day ? '这一天没有情报' : '这个月还没有收录情报'"
        />

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
        </div>
      </KunLoadingDim>

      <KunPagination
        v-if="totalPage > 1"
        v-model:current-page="page"
        :total-page="totalPage"
        :is-loading="status === 'pending'"
        :page-href="pageHref"
      />

      <NewsPartners v-if="partners.length" :sources="partners" />
    </div>
  </div>
</template>
