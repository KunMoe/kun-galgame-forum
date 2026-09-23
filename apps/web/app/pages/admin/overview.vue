<script setup lang="ts">
import { watchDebounced } from '@vueuse/core'
import {
  KUN_ADMIN_OVERVIEW_STATS_MODEL_ITEM,
  KUN_ADMIN_OVERVIEW_STATS_MODEL_MAP
} from '~/constants/admin'

definePageMeta({ middleware: 'permission', permissions: ['admin.dashboard'] })

const selectedDays = ref(7)
const debouncedDays = ref(selectedDays.value)

watchDebounced(
  selectedDays,
  (newValue) => {
    debouncedDays.value = newValue
  },
  { debounce: 300, maxWait: 1000 }
)

const [{ data: overview }, { data: daily }] = await Promise.all([
  useApi('admin-overview', (client, { signal }) =>
    client.GET('/admin/overview', { signal })
  ),
  useApi(
    () => `admin-overview-daily:${debouncedDays.value}`,
    (client, { signal }) =>
      client.GET('/admin/overview/daily', {
        params: { query: { days: debouncedDays.value } },
        signal
      })
  )
])

const days = computed(() => daily.value?.items ?? [])

const allStats = computed(() => {
  const totals = overview.value
  if (!totals) return []
  return KUN_ADMIN_OVERVIEW_STATS_MODEL_ITEM.map((key) => ({
    name: key,
    label: KUN_ADMIN_OVERVIEW_STATS_MODEL_MAP[key].label,
    count: totals[key]
  }))
})

const totalStats = computed(() =>
  KUN_ADMIN_OVERVIEW_STATS_MODEL_ITEM.map((key) => ({
    name: key,
    label: KUN_ADMIN_OVERVIEW_STATS_MODEL_MAP[key].label,
    total: days.value.reduce((sum, day) => sum + day[key], 0)
  }))
)

useKunDisableSeo('数据总览')
</script>

<template>
  <div class="space-y-12">
    <KunHeader
      name="数据总览"
      description="查看过去一段时间内话题, Galgame, Galgame 资源, Galgame 网站, Galgame 评论, Galgame 网站评论的增长趋势。好的网站理应的开放的, 您有权利知道这个网站的一切, 当天的数据将会延时更新若干小时。"
    />

    <div class="space-y-3">
      <h2 class="text-2xl">加和数据</h2>
      <p class="text-default-500 text-sm">网站在建立以来, 各项指标的总和数据</p>
      <div
        v-if="allStats.length"
        class="grid grid-cols-2 gap-3 lg:grid-cols-3 xl:grid-cols-5"
      >
        <KunCard
          :is-transparent="true"
          :is-hoverable="false"
          v-for="stat in allStats"
          :key="stat.name"
        >
          <p
            class="truncate text-sm font-medium text-gray-500 dark:text-gray-400"
          >
            {{ stat.label }}
          </p>
          <p class="text-2xl font-extrabold text-gray-900 dark:text-white">
            {{ stat.count.toLocaleString() }}
          </p>
        </KunCard>
      </div>
    </div>

    <div class="space-y-3">
      <h2 class="text-2xl">增量数据</h2>
      <p class="text-default-500 text-sm">
        网站在指定时间段内, 各项指标的增量数据
      </p>
      <KunSlider :min="1" :max="30" :step="1" v-model="selectedDays" />
      <div class="grid grid-cols-2 gap-3 lg:grid-cols-3 xl:grid-cols-5">
        <KunCard
          :is-transparent="true"
          :is-hoverable="false"
          v-for="stat in totalStats"
          :key="stat.name"
        >
          <p
            class="truncate text-sm font-medium text-gray-500 dark:text-gray-400"
          >
            {{ stat.label }}
          </p>
          <p
            class="flex items-center gap-3 text-2xl font-extrabold text-gray-900 dark:text-white"
          >
            <KunIcon
              v-if="stat.total > 0"
              name="lucide:trending-up"
              class-name="text-success-600"
            />
            <span>{{ stat.total.toLocaleString() }}</span>
          </p>
          <p class="mt-1 text-xs text-gray-400 dark:text-gray-500">
            最近 {{ selectedDays }} 天新增
          </p>
        </KunCard>
      </div>
    </div>

    <div class="space-y-3">
      <h2 class="text-2xl">增量趋势图</h2>
      <p class="text-default-500 text-sm">
        如果要用一张可视化的图表来表示网站的增量数据状态, 那就是下面这张图
      </p>
      <KunCard :is-transparent="true" :is-hoverable="false">
        <AdminOverviewChart v-if="days.length" :data="days" />
      </KunCard>
    </div>
  </div>
</template>
