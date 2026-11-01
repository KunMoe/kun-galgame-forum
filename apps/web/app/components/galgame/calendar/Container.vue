<script setup lang="ts">
import type { KunTabItem } from '@kungal/ui-vue'
import type {
  ReleaseCalendarMonth,
  ReleaseCalendarPending,
  ReleaseCalendarTBA,
  ReleaseCalendarUpcoming,
  WorkSummary
} from '#shared/utils/api/schemas'
import { workSummaryToCard } from '~/utils/galgame/workCard'

const { view, month, year, doujin, followed } = useQueryState({
  view: queryString('month'),
  month: queryString(),
  year: queryString(),
  doujin: queryString(),
  followed: queryString()
})
const includeDoujin = computed({
  get: () => doujin.value === '1',
  set: (v: boolean) => {
    doujin.value = v ? '1' : ''
  }
})
const includeFollowed = computed({
  get: () => followed.value === '1',
  set: (v: boolean) => {
    followed.value = v ? '1' : ''
  }
})
const { id: currentUserId } = storeToRefs(usePersistUserStore())
const followedOnly = computed(
  () => !!currentUserId.value && includeFollowed.value
)
const { allowsNsfw, stanceKey } = useContentStance()
const { excludeQuery } = useTagBlocks()
const blockKey = computed(() => JSON.stringify(excludeQuery.value))
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
  problem: monthProblem,
  pending: monthLoading
} = useKunApi<ReleaseCalendarMonth>(
  () =>
    `release-calendar:${month.value}:${stanceKey.value}:${includeDoujin.value}:${followedOnly.value}:${blockKey.value}`,
  (api, { signal }) => {
    if (followedOnly.value) {
      return api.GET('/me/release-calendar', {
        params: {
          query: {
            ...(month.value ? { month: month.value } : {}),
            include_nsfw: allowsNsfw.value,
            ...excludeQuery.value
          }
        },
        signal
      })
    }
    return api.GET('/release-calendar', {
      params: {
        query: {
          ...(month.value ? { month: month.value } : {}),
          include_nsfw: allowsNsfw.value,
          include_doujin: includeDoujin.value,
          ...excludeQuery.value
        }
      },
      signal
    })
  },
  { immediate: isMonthView.value, server: isMonthView.value }
)

const {
  data: upcomingData,
  status: upcomingStatus,
  refresh: refreshUpcoming,
  problem: upcomingProblem,
  pending: upcomingLoading
} = useKunApi<ReleaseCalendarUpcoming>(
  () =>
    `release-calendar-upcoming:${stanceKey.value}:${includeDoujin.value}:${followedOnly.value}:${blockKey.value}`,
  (api, { signal }) => {
    if (followedOnly.value) {
      return api.GET('/me/release-calendar/upcoming', {
        params: {
          query: { include_nsfw: allowsNsfw.value, ...excludeQuery.value }
        },
        signal
      })
    }
    return api.GET('/release-calendar/upcoming', {
      params: {
        query: {
          include_nsfw: allowsNsfw.value,
          include_doujin: includeDoujin.value,
          ...excludeQuery.value
        }
      },
      signal
    })
  },
  { immediate: view.value === 'upcoming', server: view.value === 'upcoming' }
)

const {
  data: pendingData,
  status: pendingStatus,
  refresh: refreshPending,
  problem: pendingProblem,
  pending: pendingLoading
} = useKunApi<ReleaseCalendarPending>(
  () =>
    `release-calendar-pending:${year.value}:${stanceKey.value}:${includeDoujin.value}:${followedOnly.value}:${blockKey.value}`,
  (api, { signal }) => {
    if (followedOnly.value) {
      return api.GET('/me/release-calendar/pending', {
        params: {
          query: {
            ...(year.value ? { year: Number(year.value) } : {}),
            include_nsfw: allowsNsfw.value,
            ...excludeQuery.value
          }
        },
        signal
      })
    }
    return api.GET('/release-calendar/pending', {
      params: {
        query: {
          ...(year.value ? { year: Number(year.value) } : {}),
          include_nsfw: allowsNsfw.value,
          include_doujin: includeDoujin.value,
          ...excludeQuery.value
        }
      },
      signal
    })
  },
  { immediate: view.value === 'pending', server: view.value === 'pending' }
)

const {
  data: tbaData,
  status: tbaStatus,
  refresh: refreshTba,
  problem: tbaProblem,
  pending: tbaLoading
} = useKunApi<ReleaseCalendarTBA>(
  () =>
    `release-calendar-tba:${stanceKey.value}:${includeDoujin.value}:${followedOnly.value}:${blockKey.value}`,
  (api, { signal }) => {
    if (followedOnly.value) {
      return api.GET('/me/release-calendar/tba', {
        params: {
          query: { include_nsfw: allowsNsfw.value, ...excludeQuery.value }
        },
        signal
      })
    }
    return api.GET('/release-calendar/tba', {
      params: {
        query: {
          include_nsfw: allowsNsfw.value,
          include_doujin: includeDoujin.value,
          ...excludeQuery.value
        }
      },
      signal
    })
  },
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

const focusDate = ref<string | null>(null)

const goMonth = (m: string) => {
  focusDate.value = null
  month.value = m === monthData.value?.today_date.slice(0, 7) ? '' : m
}
const goDate = (date: string) => {
  goMonth(date.slice(0, 7))
  focusDate.value = date
}
const goPrevMonth = () => {
  if (monthData.value?.has_prev && monthData.value.prev_month) {
    goMonth(monthData.value.prev_month)
  }
}
const goNextMonth = () => {
  if (monthData.value?.has_next && monthData.value.next_month) {
    goMonth(monthData.value.next_month)
  }
}
const goToday = () => {
  goMonth('')
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
      description="按发售月份浏览即将与已发售的 Galgame。数据来自 鲲 Galgame 论坛, 月份边界按日本时间 (JST) 计。打开任意作品即可发布资源。"
    />

    <KunTab :items="tabs" v-model="view" variant="bordered" />

    <KunSwitch
      v-if="!followedOnly"
      v-model="includeDoujin"
      label="包含同人作品"
      description="默认只显示原语言为日语的商业作品, 开启后加入同人社团的作品"
    />

    <KunSwitch
      v-if="currentUserId"
      v-model="includeFollowed"
      label="只看我关注的厂商"
    />

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

      <GalgameCardGridSkeleton v-if="upcomingLoading" />

      <KunLoading v-else :loading="upcomingStatus === 'pending'">
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
              <p v-if="grp.is_truncated" class="text-default-400 text-sm">
                本月作品过多，未全部列出
              </p>
              <GalgameCard :galgames="cardsOf(grp.items)" />
            </section>
          </div>
          <div
            v-else-if="followedOnly"
            class="flex flex-col items-center gap-3"
          >
            <KunNull description="还没有关注的厂商有作品在这里" />
            <KunLink to="/galgame/official" size="sm" underline="hover">
              去厂商页关注
            </KunLink>
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

      <GalgameCardGridSkeleton v-if="pendingLoading" />

      <KunLoading v-else :loading="pendingStatus === 'pending'">
        <template v-if="pendingData">
          <GalgameCard
            v-if="pendingData.items.length"
            :galgames="cardsOf(pendingData.items)"
          />
          <div
            v-else-if="followedOnly"
            class="flex flex-col items-center gap-3"
          >
            <KunNull description="还没有关注的厂商有作品在这里" />
            <KunLink to="/galgame/official" size="sm" underline="hover">
              去厂商页关注
            </KunLink>
          </div>
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

      <GalgameCardGridSkeleton v-if="tbaLoading" />

      <KunLoading v-else :loading="tbaStatus === 'pending'">
        <template v-if="tbaData">
          <GalgameCard
            v-if="tbaData.items.length"
            :galgames="cardsOf(tbaData.items)"
          />
          <div
            v-else-if="followedOnly"
            class="flex flex-col items-center gap-3"
          >
            <KunNull description="还没有关注的厂商有作品在这里" />
            <KunLink to="/galgame/official" size="sm" underline="hover">
              去厂商页关注
            </KunLink>
          </div>
          <KunNull v-else description="暂无发售日期未定的作品" />
        </template>
      </KunLoading>
    </template>

    <template v-else>
      <div v-if="monthLoading" class="flex flex-col gap-3">
        <KunSkeleton height="3.25rem" rounded="lg" />
        <GalgameCardGridSkeleton />
      </div>

      <KunLoading v-else :loading="monthStatus === 'pending'">
        <template v-if="monthData">
          <GalgameCalendarMonth
            v-if="monthData.items.length || !followedOnly"
            :data="monthData"
            :focus-date="focusDate"
            @prev="goPrevMonth"
            @next="goNextMonth"
            @today="goToday"
            @jump-month="goMonth"
            @jump-date="goDate"
          />
          <div v-else class="flex flex-col gap-3">
            <div
              class="border-default-200 flex items-center justify-between gap-2 rounded-xl border px-2 py-1.5"
            >
              <KunButton
                variant="light"
                :is-icon-only="true"
                :disabled="!monthData.has_prev"
                @click="goPrevMonth"
              >
                <KunIcon name="lucide:chevron-left" class="size-5" />
              </KunButton>
              <div class="flex flex-col items-center">
                <span class="font-bold">
                  {{ ymLabel(monthData.calendar_month) }}
                </span>
                <span class="text-default-400 text-xs">
                  共 {{ monthData.item_count }} 部
                </span>
              </div>
              <KunButton
                variant="light"
                :is-icon-only="true"
                :disabled="!monthData.has_next"
                @click="goNextMonth"
              >
                <KunIcon name="lucide:chevron-right" class="size-5" />
              </KunButton>
            </div>
            <GalgameCalendarJump
              :data="monthData"
              @month="goMonth"
              @date="goDate"
            />
            <div
              v-if="
                monthData.calendar_month !== monthData.today_date.slice(0, 7)
              "
              class="flex justify-center"
            >
              <KunButton variant="light" size="sm" @click="goToday">
                <KunIcon name="lucide:undo-2" class="size-4" />
                回到本月
              </KunButton>
            </div>
            <div class="flex flex-col items-center gap-3">
              <KunNull description="还没有关注的厂商有作品在这里" />
              <KunLink to="/galgame/official" size="sm" underline="hover">
                去厂商页关注
              </KunLink>
            </div>
          </div>
        </template>
      </KunLoading>
    </template>
  </div>
</template>
