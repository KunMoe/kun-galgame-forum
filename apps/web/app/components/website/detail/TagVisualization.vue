<script setup lang="ts">
import type { WebsiteTag } from '#shared/utils/api/schemas'

const props = defineProps<{
  tags: WebsiteTag[]
}>()

const isShowDetail = ref(false)

const { data: groups } = useWebsiteTagGroups()
const { data: allTags } = useWebsiteTags()

const categoryThemeColors = [
  'primary',
  'success',
  'secondary',
  'info',
  'default',
  'warning',
  'danger'
]

const totalScore = computed(() => {
  return props.tags.reduce((sum, tag) => sum + (tag.level || 0), 0)
})

const UNGROUPED = 'ungrouped'

// The best score a site can reach with today's tags: the top level of each
// single-select group plus every positive level elsewhere. It was hard-coded
// as 245 while the taxonomy allowed 275.
const maxScore = computed(() => {
  const multi = new Map(
    (groups.value ?? []).map((group) => [group.id, group.is_multi_select])
  )
  const best = new Map<string, number>()
  let loose = 0
  for (const tag of allTags.value ?? []) {
    const level = Math.max(tag.level, 0)
    const groupId = tag.website_tag_group_id
    if (!groupId || multi.get(groupId) !== false) {
      loose += level
      continue
    }
    best.set(groupId, Math.max(best.get(groupId) ?? 0, level))
  }
  const total = [...best.values()].reduce((sum, level) => sum + level, loose)
  return Math.max(total, 1)
})

const categoryStats = computed(() => {
  const labels = new Map<string, string>(
    (groups.value ?? []).map((group) => [group.id, group.label || group.slug])
  )

  const categories: Record<
    string,
    {
      name: string
      label: string
      score: number
      tags: WebsiteTag[]
      color: string
    }
  > = {}

  props.tags.forEach((tag) => {
    const groupId = tag.website_tag_group_id ?? UNGROUPED
    const score = tag.level || 0

    if (!categories[groupId]) {
      categories[groupId] = {
        name: groupId,
        label: labels.get(groupId) ?? '其他指标',
        score: 0,
        tags: [],
        color:
          categoryThemeColors[
            Object.keys(categories).length % categoryThemeColors.length
          ]!
      }
    }

    categories[groupId].score += score
    categories[groupId].tags.push({ ...tag, label: tag.label || tag.slug })
  })

  const sortedCategories = Object.values(categories).sort(
    (a, b) => b.score - a.score
  )
  const totalAbsScore = sortedCategories.reduce(
    (sum, cat) => sum + Math.abs(cat.score),
    0
  )

  let currentPosition = 0
  return sortedCategories.map((category) => {
    const width =
      totalAbsScore > 0 ? (Math.abs(category.score) / totalAbsScore) * 100 : 0
    const result = {
      ...category,
      position: currentPosition,
      width
    }
    currentPosition += width
    return result
  })
})

const getScoreColor = (score: number): string => {
  if (score >= 100) return 'text-success-600'
  if (score >= 50) return 'text-default-500'
  if (score >= 0) return 'text-warning-500'
  return 'text-danger-500'
}

const getBackgroundColor = (color: string): string => {
  if (color === 'primary') return 'bg-primary-500/30'
  if (color === 'secondary') return 'bg-primary-500/30'
  if (color === 'success') return 'bg-success-500/30'
  if (color === 'info') return 'bg-info-500/30'
  if (color === 'warning') return 'bg-warning-500/30'
  return 'bg-danger-500/30'
}
</script>

<template>
  <div class="space-y-6">
    <div>
      <div class="mb-4 flex items-center justify-between">
        <h2 class="text-foreground text-2xl font-bold">网站价值精算值</h2>
        <div class="flex items-center space-x-4">
          <div class="text-3xl font-bold" :class="getScoreColor(totalScore)">
            {{ `${totalScore} / ${maxScore}` }}
          </div>
        </div>
      </div>

      <div
        :class="
          cn(
            'mb-2 overflow-hidden rounded-full',
            totalScore > 0 ? 'bg-content3' : 'bg-danger-500/30'
          )
        "
      >
        <div
          class="relative h-8"
          :style="{
            width: ((totalScore > 0 ? totalScore : 0) / maxScore) * 100 + '%'
          }"
        >
          <div
            v-for="(category, index) in categoryStats"
            :key="index"
            :class="
              cn(
                'absolute top-0 h-full transition-all duration-500',
                getBackgroundColor(category.color)
              )
            "
            :style="{
              left: category.position + '%',
              width: category.width + '%'
            }"
          />
        </div>
      </div>

      <div class="mt-3 flex flex-wrap gap-2">
        <div
          v-for="category in categoryStats"
          :key="category.name"
          class="flex items-center space-x-2 text-sm"
        >
          <div
            :class="
              cn('h-3 w-3 rounded-full', getBackgroundColor(category.color))
            "
          />
          <span class="font-medium">{{ category.label }}</span>
          <span class="text-default-400">({{ category.score }})</span>
        </div>
      </div>
    </div>

    <div
      v-if="isShowDetail"
      class="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3"
    >
      <KunCard
        :is-transparent="true"
        :is-hoverable="true"
        v-for="category in categoryStats"
        :key="category.name"
        content-class="justify-start"
      >
        <div class="mb-3 flex items-center justify-between">
          <h3 class="text-foreground font-semibold">
            {{ category.label }}
          </h3>
          <span
            class="rounded px-2 py-1 text-sm font-medium"
            :class="[`text-${category.color}`, `bg-${category.color}/20`]"
          >
            {{ category.score }}
          </span>
        </div>

        <div class="space-y-2">
          <div
            v-for="tag in category.tags"
            :key="tag.id"
            class="flex flex-col justify-between rounded border p-2"
          >
            <div class="text-sm font-medium">
              {{ tag.label }}
            </div>

            <div class="text-default-500 my-1 text-xs">
              {{ tag.description }}
            </div>

            <div class="flex items-center space-x-2">
              <div class="bg-content3 h-2 w-32 overflow-hidden rounded-full">
                <div
                  class="h-full transition-all duration-300"
                  :class="tag.level >= 0 ? 'bg-success' : 'bg-danger'"
                  :style="{
                    width: (Math.abs(tag.level) / 20) * 100 + '%'
                  }"
                />
              </div>
              <span
                class="text-sm font-medium"
                :class="tag.level >= 0 ? 'text-success-600' : 'text-danger-600'"
              >
                {{ tag.level >= 0 ? '+' : '' }}{{ tag.level }}
              </span>
            </div>
          </div>
        </div>
      </KunCard>
    </div>

    <KunInfo
      title="评分说明"
      color="info"
      description="评分基于网站的各项指标, 包括广告程度、资源数量、更新频率等。正分表示优质特性, 负分表示存在问题。总分越高表示网站质量越好, 超过 100 分的网站已经很不错了。"
    >
      <p class="text-info-800 text-sm">
        评分数值仅供参考, 如果您对此评分或者网站数据有任何问题, 欢迎评论或
        <KunLink size="sm" to="/doc/contact"> 联系我们 </KunLink>。
      </p>
      <KunButton variant="flat" size="sm" @click="isShowDetail = !isShowDetail">
        展开网站价值分析详情
      </KunButton>
    </KunInfo>
  </div>
</template>
