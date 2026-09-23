<script setup lang="ts">
import { useRouteQuery } from '@vueuse/router'
import type { KunTabItem } from '@kungal/ui-vue'
import type {
  ReleaseCalendarMonth,
  ReleaseCalendarPending,
  ReleaseCalendarTBA,
  ReleaseCalendarUpcoming,
  WorkSummary
} from '#shared/utils/api/schemas'
import { workSummaryToCard } from '~/utils/galgame/workCard'

const opts = { mode: 'replace' as const }
const view = useRouteQuery<string>('view', 'month', opts)
const month = useRouteQuery<string>('month', '', opts)
const year = useRouteQuery<string>('year', '', opts)
const { allowsNsfw } = useContentStance()
const nameOf = useCatalogName()

const tabs: KunTabItem[] = [
  { value: 'month', textValue: '月历' },
  { value: 'upcoming', textValue: '未发售' },
  { value: 'pending', textValue: '年内待定' },
  { value: 'tba', textValue: '发售日期未定' }
]

const isMonthView = computed(
  () =>
    view.value !== 'upcoming' &&
    view.value !== 'pending' &&
    view.value !== 'tba'
)

const cardsOf = (items: WorkSummary[]) =>
  items.map((item) => workSummaryToCard(item, nameOf))

const {
  data: monthData,
  status: monthStatus,
  refresh: refreshMonth,
  problem: monthProblem
} = await useApi<ReleaseCalendarMonth>(
  () => `release-calendar:${month.value}:${allowsNsfw.value}`,
  (api, { signal }) =>
    api.GET('/release-calendar', {
      params: {
        query: {
          ...(month.value ? { month: month.value } : {}),
          include_nsfw: allowsNsfw.value
        }
      },
      signal
    }),
  { immediate: isMonthView.value, server: isMonthView.value }
)

const {
  data: upcomingData,
  status: upcomingStatus,
  refresh: refreshUpcoming,
  problem: upcomingProblem
} = await useApi<ReleaseCalendarUpcoming>(
  () => `release-calendar-upcoming:${allowsNsfw.value}`,
  (api, { signal }) =>
    api.GET('/release-calendar/upcoming', {
      params: { query: { include_nsfw: allowsNsfw.value } },
      signal
    }),
  { immediate: view.value === 'upcoming', server: view.value === 'upcoming' }
)

const {
  data: pendingData,
  status: pendingStatus,
  refresh: refreshPending,
  problem: pendingProblem
} = await useApi<ReleaseCalendarPending>(
  () => `release-calendar-pending:${year.value}:${allowsNsfw.value}`,
  (api, { signal }) =>
    api.GET('/release-calendar/pending', {
      params: {
        query: {
          ...(year.value ? { year: Number(year.value) } : {}),
          include_nsfw: allowsNsfw.value
        }
      },
      signal
    }),
  { immediate: view.value === 'pending', server: view.value === 'pending' }
)

const {
  data: tbaData,
  status: tbaStatus,
  refresh: refreshTba,
  problem: tbaProblem
} = await useApi<ReleaseCalendarTBA>(
  () => `release-calendar-tba:${allowsNsfw.value}`,
  (api, { signal }) =>
    api.GET('/release-calendar/tba', {
      params: { query: { include_nsfw: allowsNsfw.value } },
      signal
    }),
  { immediate: view.value === 'tba', server: view.value === 'tba' }
)

watch(monthProblem, (p) => {
  if (p) reportProblem(p)
})
watch(upcomingProblem, (p) => {
  if (p) reportProblem(p)
})
watch(pendingProblem, (p) => {
  if (p) reportProblem(p)
})
watch(tbaProblem, (p) => {
  if (p) reportProblem(p)
})

watch(view, () => {
  if (isMonthView.value && !monthData.value) {
    refreshMonth()
  } else if (view.value === 'upcoming' && !upcomingData.value) {
    refreshUpcoming()
  } else if (view.value === 'pending' && !pendingData.value) {
    refreshPending()
  } else if (view.value === 'tba' && !tbaData.value) {
    refreshTba()
  }
})

const ymLabel = (m: string) => {
  const [y, mo] = m.split('-')
  return `${y} 年 ${Number(mo)} 月`
}

const goPrevMonth = () => {
  if (monthData.value?.has_prev && monthData.value.prev_month) {
    month.value = monthData.value.prev_month
  }
}
const goNextMonth = () => {
  if (monthData.value?.has_next && monthData.value.next_month) {
    month.value = monthData.value.next_month
  }
}
const goToday = () => {
  month.value = ''
}

const goPrevYear = () => {
  const y = Number(pendingData.value?.year)
  if (y) {
    year.value = String(y - 1)
  }
}
const goNextYear = () => {
  const y = Number(pendingData.value?.year)
  if (y) {
    year.value = String(y + 1)
  }
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <KunHeader
      name="Galgame 发售月历"
      description="按发售月份浏览即将与已发售的 Galgame。数据来自 Galgame 资料库, 月份边界按日本时间 (JST) 计。打开任意作品即可发布资源。"
    />

    <KunTab :items="tabs" v-model="view" variant="bordered" />

    <template v-if="view === 'upcoming'">
      <div
        class="border-default-200 flex items-center justify-center gap-2 rounded-xl border px-3 py-3"
      >
        <KunIcon name="lucide:calendar-clock" class="text-default-400 size-5" />
        <span class="font-medium">未发售 · 已定档排期</span>
        <span class="text-default-400 text-sm">
          共 {{ upcomingData?.item_count ?? 0 }} 部
        </span>
      </div>

      <KunLoading :loading="upcomingStatus === 'pending'">
        <template v-if="upcomingData">
          <div v-if="upcomingData.entries.length" class="flex flex-col gap-6">
            <section
              v-for="grp in upcomingData.entries"
              :key="grp.calendar_month"
              class="flex flex-col gap-2"
            >
              <div class="flex items-center gap-2">
                <KunIcon
                  name="lucide:calendar"
                  class="text-default-500 size-4"
                />
                <h3 class="font-medium">{{ ymLabel(grp.calendar_month) }}</h3>
                <span class="text-default-400 text-sm">
                  {{ grp.items.length }} 部
                </span>
              </div>
              <p
                v-if="grp.is_truncated"
                class="text-default-400 text-sm"
              >
                本月作品过多，未全部列出
              </p>
              <GalgameCard :galgames="cardsOf(grp.items)" />
            </section>
          </div>
          <KunNull v-else description="暂无已定档的未发售作品" />
        </template>
      </KunLoading>
    </template>

    <template v-else-if="view === 'pending'">
      <div
        class="border-default-200 flex items-center justify-between gap-2 rounded-xl border px-3 py-2"
      >
        <KunButton variant="light" :is-icon-only="true" @click="goPrevYear">
          <KunIcon name="lucide:chevron-left" class="size-5" />
        </KunButton>
        <div class="flex flex-col items-center">
          <span class="text-xl font-bold sm:text-2xl">
            {{ pendingData?.year }} 年
          </span>
          <span class="text-default-400 text-xs">
            仅知年份 · 月份待定 · 共 {{ pendingData?.item_count ?? 0 }} 部
          </span>
        </div>
        <KunButton variant="light" :is-icon-only="true" @click="goNextYear">
          <KunIcon name="lucide:chevron-right" class="size-5" />
        </KunButton>
      </div>

      <p
        v-if="pendingData?.is_truncated"
        class="text-default-400 text-center text-sm"
      >
        作品过多，未全部列出
      </p>

      <KunLoading :loading="pendingStatus === 'pending'">
        <template v-if="pendingData">
          <GalgameCard
            v-if="pendingData.items.length"
            :galgames="cardsOf(pendingData.items)"
          />
          <KunNull v-else description="该年暂无仅知年份的待定作品" />
        </template>
      </KunLoading>
    </template>

    <template v-else-if="view === 'tba'">
      <div
        class="border-default-200 flex items-center justify-center gap-2 rounded-xl border px-3 py-3"
      >
        <KunIcon name="lucide:calendar-x" class="text-default-400 size-5" />
        <span class="font-medium">发售日期未定</span>
        <span class="text-default-400 text-sm">
          共 {{ tbaData?.item_count ?? 0 }} 部
        </span>
      </div>

      <p
        v-if="tbaData?.is_truncated"
        class="text-default-400 text-center text-sm"
      >
        作品过多，未全部列出
      </p>

      <KunLoading :loading="tbaStatus === 'pending'">
        <template v-if="tbaData">
          <GalgameCard
            v-if="tbaData.items.length"
            :galgames="cardsOf(tbaData.items)"
          />
          <KunNull v-else description="暂无发售日期未定的作品" />
        </template>
      </KunLoading>
    </template>

    <template v-else>
      <KunLoading :loading="monthStatus === 'pending'">
        <GalgameCalendarMonth
          v-if="monthData"
          :data="monthData"
          @prev="goPrevMonth"
          @next="goNextMonth"
          @today="goToday"
        />
      </KunLoading>
    </template>
  </div>
</template>
