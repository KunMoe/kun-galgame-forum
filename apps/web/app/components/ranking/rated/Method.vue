<script setup lang="ts">
withDefaults(defineProps<{ fittedAt?: string | null }>(), { fittedAt: null })

const POINTS = [
  {
    icon: 'lucide:gauge',
    title: '综合评分是偏差值',
    text: '50 分是被两个以上站点评过的典型作品, 每 10 分为一个标准差, 60 分以上就明显好于平均。VNDB 的 10 分制、DLsite 的 5 星制和批评空间的 100 分制先各自换算到站内位置, 再校正各站受众的口味差异后合并。'
  },
  {
    icon: 'lucide:ruler',
    title: '按置信区间的下端排名',
    text: '票数越少、评过的站点越少, 区间就越宽。默认按 90% 区间的下端排名, 三张满分票排不到三千票的名作前面; 地雷榜则按区间上端从低到高排。'
  },
  {
    icon: 'lucide:list-checks',
    title: '各站原分原样展示',
    text: '每部作品下列出的是各站自己公布的分数, 没有改动。「估计」是把作品自身的均值向同等票数作品的典型分数收缩后的真实水平估计。'
  },
  {
    icon: 'lucide:refresh-cw',
    title: '每日更新',
    text: '评分模型与排行每夜重新计算, 新评分最晚第二天进榜。默认只排视觉小说, 在「作品范围」里可以加入主机与 PC 游戏。数据由 NextMoe 开放 API 提供。'
  }
]
</script>

<template>
  <KunPopover
    position="bottom-end"
    aria-label="评分说明"
    inner-class="w-96 max-w-[calc(100vw-2rem)] p-3"
  >
    <template #trigger>
      <KunTooltip text="评分说明" position="bottom">
        <button
          type="button"
          aria-label="评分说明"
          :class="filterPillSquareClass(false)"
        >
          <KunIcon name="lucide:info" class="size-4 text-inherit" />
        </button>
      </KunTooltip>
    </template>

    <ul class="space-y-3">
      <li v-for="point in POINTS" :key="point.title" class="flex gap-2.5">
        <KunIcon
          :name="point.icon"
          class="text-primary mt-0.5 size-4 shrink-0"
        />
        <div class="space-y-0.5">
          <p class="text-sm font-medium">{{ point.title }}</p>
          <p class="text-default-500 text-xs leading-relaxed">
            {{ point.text }}
          </p>
        </div>
      </li>
    </ul>
    <p
      v-if="fittedAt"
      class="border-default-200 text-default-400 mt-3 border-t pt-2 text-xs"
    >
      当前模型计算于
      {{ formatDate(fittedAt, { isShowYear: true, isPrecise: true }) }}
    </p>
  </KunPopover>
</template>
