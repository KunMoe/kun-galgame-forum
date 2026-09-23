<script setup lang="ts">
import VueApexCharts from 'vue3-apexcharts'
import type { ApexOptions } from 'apexcharts'
import {
  KUN_ADMIN_OVERVIEW_STATS_MODEL_ITEM,
  KUN_ADMIN_OVERVIEW_STATS_MODEL_MAP
} from '~/constants/admin'
import type { OverviewDay } from '#shared/utils/api/schemas'

const props = defineProps<{
  data: OverviewDay[]
}>()

const colorMode = useColorMode()

const chartSeries = computed(() =>
  KUN_ADMIN_OVERVIEW_STATS_MODEL_ITEM.map((key) => ({
    name: KUN_ADMIN_OVERVIEW_STATS_MODEL_MAP[key].label,
    data: props.data.map((d) => d[key])
  }))
)

const chartOptions = computed<ApexOptions>(() => {
  const categories = props.data.map((d) => d.bucket_date)
  const colors = Object.values(KUN_ADMIN_OVERVIEW_STATS_MODEL_MAP).map(
    (m) => m.color
  )

  return {
    chart: {
      type: 'area',
      height: 400,
      zoom: { enabled: false },
      toolbar: { show: false },
      fontFamily: 'inherit'
    },
    colors: colors,
    dataLabels: { enabled: false },
    stroke: { curve: 'smooth', width: 2 },
    xaxis: {
      type: 'datetime',
      categories: categories,
      labels: {
        style: {
          colors: 'var(--color-default-500)'
        }
      },
      axisBorder: {
        color: 'var(--color-default-200)'
      },
      axisTicks: {
        color: 'var(--color-default-200)'
      }
    },
    yaxis: {
      labels: {
        style: {
          colors: 'var(--color-default-500)'
        }
      }
    },
    legend: {
      position: 'top',
      horizontalAlign: 'right',
      labels: {
        colors: 'var(--color-default-500)'
      }
    },
    // OUTSIDE 铁律 #1 (no gradients): this is an ApexCharts area series fill —
    // the fade under a plotted line, not a UI background. The rule's three
    // listed exceptions are the only ones in UI markup; a `rg gradient` sweep
    // hits this fourth file too. Do NOT flatten it to a solid colour.
    fill: {
      type: 'gradient',
      gradient: {
        shadeIntensity: 1,
        opacityFrom: 0.4,
        opacityTo: 0.1,
        stops: [0, 90, 100]
      }
    },
    tooltip: {
      x: { format: 'yyyy/MM/dd' },
      theme: colorMode.value
    },
    grid: {
      borderColor: 'var(--color-default-200)',
      strokeDashArray: 4
    }
  }
})

const showChart = ref(false)
const nuxtApp = useNuxtApp()

onMounted(() => {
  nuxtApp.hook('page:transition:finish', () => {
    showChart.value = true
  })
})
</script>

<template>
  <ClientOnly>
    <template v-if="showChart">
      <VueApexCharts
        type="area"
        height="400"
        :options="chartOptions"
        :series="chartSeries"
      />
    </template>
    <template #fallback>
      <KunLoading description="正在加载图表..." />
    </template>
  </ClientOnly>
</template>
