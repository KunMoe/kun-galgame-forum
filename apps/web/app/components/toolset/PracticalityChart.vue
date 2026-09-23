<script setup lang="ts">
import VueApexCharts from 'vue3-apexcharts'
import type { ApexOptions } from 'apexcharts'

const props = defineProps<{
  id: string
  distribution: number[]
  average: number | null
  rating: number | null
}>()

const colorMode = useColorMode()

const categories = computed(() =>
  Array.from({ length: 5 }, (_, i) => String(i + 1))
)

const countsArray = computed(() =>
  categories.value.map((_, i) => props.distribution[i] ?? 0)
)

const mineIndex = computed(() =>
  props.rating == null ? -1 : props.rating - 1
)

const series = computed(() => [{ name: '评分人数', data: countsArray.value }])

const barColors = computed(() =>
  categories.value.map((_, i) =>
    i === mineIndex.value ? 'var(--color-warning)' : 'var(--color-primary)'
  )
)

const options = computed(
  (): ApexOptions => ({
    chart: {
      id: `practicality-chart-${props.id}`,
      type: 'bar',
      height: 260,
      toolbar: { show: false },
      fontFamily: 'inherit'
    },
    plotOptions: {
      bar: {
        horizontal: true,
        distributed: true,
        borderRadius: 4,
        barHeight: '60%'
      }
    },
    dataLabels: {
      enabled: true,
      formatter: (val) => String(val ?? ''),
      style: { fontSize: '12px', colors: ['var(--color-default-900)'] }
    },
    colors: barColors.value,
    xaxis: {
      categories: categories.value.map((c) => `${c} 星`),
      labels: { style: { colors: 'var(--color-default-500)' } }
    },
    yaxis: {
      labels: { style: { colors: 'var(--color-default-900)' } }
    },
    tooltip: {
      theme: colorMode.value,
      y: { formatter: (y: number) => `${y} 人` }
    },
    grid: { borderColor: 'var(--color-default-200)', strokeDashArray: 4 },
    legend: { show: false }
  })
)
</script>

<template>
  <div class="contents">
    <ClientOnly>
      <div class="space-y-2">
        <div class="flex items-center gap-3">
          <h3 class="font-semibold">工具实用性分布</h3>
          <span class="text-default-500">
            平均
            <span class="text-warning text-lg font-bold">
              {{ average == null ? '暂无' : average.toFixed(1) }}
            </span>
            <template v-if="average != null"> / 5.0</template>
          </span>
        </div>

        <VueApexCharts
          :key="rating ?? 0"
          type="bar"
          height="260"
          :options="options"
          :series="series"
        />
      </div>
    </ClientOnly>
  </div>
</template>
