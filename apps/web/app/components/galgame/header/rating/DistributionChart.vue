<script setup lang="ts">
import VueApexCharts from 'vue3-apexcharts'
import type { ApexOptions } from 'apexcharts'

const props = withDefaults(
  defineProps<{
    workId: number
    source: string
    buckets: number[]
    categories: string[]
    mineIndex?: number
    unit?: string
  }>(),
  { mineIndex: -1, unit: '分' }
)

const colorMode = useColorMode()

const series = computed(() => [{ name: '评分人数', data: props.buckets }])

const colors = computed(() =>
  props.buckets.map((_, index) =>
    index === props.mineIndex ? 'var(--color-warning)' : 'var(--color-primary)'
  )
)

const options = computed(
  (): ApexOptions => ({
    chart: {
      id: `galgame-rating-distribution-${props.source}-${props.workId}`,
      type: 'bar',
      height: 200,
      toolbar: { show: false },
      fontFamily: 'inherit'
    },
    plotOptions: {
      bar: { distributed: true, borderRadius: 4, columnWidth: '55%' }
    },
    dataLabels: { enabled: false },
    colors: colors.value,
    xaxis: {
      categories: props.categories,
      axisTicks: { show: false },
      labels: { style: { colors: 'var(--color-default-500)' } }
    },
    yaxis: {
      labels: {
        formatter: (value: number) => String(Math.round(value)),
        style: { colors: 'var(--color-default-500)' }
      }
    },
    tooltip: {
      // colorMode.preference is 'system' by default, which apexcharts does not
      // recognize: it then adds neither apexcharts-theme-light nor -dark, and
      // its only tooltip background lives on those two classes — the tooltip
      // rendered fully transparent. colorMode.value is always resolved.
      theme: colorMode.value,
      x: { formatter: (x: string) => `${x} ${props.unit}` },
      y: { formatter: (y: number) => `${y} 人` }
    },
    grid: { borderColor: 'var(--color-default-200)', strokeDashArray: 4 },
    legend: { show: false }
  })
)
</script>

<template>
  <ClientOnly>
    <VueApexCharts
      type="bar"
      height="200"
      :options="options"
      :series="series"
    />
    <template #fallback>
      <div class="h-[200px]" />
    </template>
  </ClientOnly>
</template>
