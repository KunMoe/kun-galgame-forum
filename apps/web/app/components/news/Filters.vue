<script setup lang="ts">
import type { KunTabItem } from '@kungal/ui-vue'

const props = withDefaults(
  defineProps<{
    sources: KunNewsSource[]
    archive?: KunNewsArchive | null
    orientation?: 'vertical' | 'horizontal'
  }>(),
  { archive: null, orientation: 'vertical' }
)

const lane = defineModel<string>('lane', { required: true })
const source = defineModel<string>('source', { required: true })
const year = defineModel<number>('year', { required: true })
const month = defineModel<number>('month', { required: true })

const isRail = computed(() => props.orientation === 'vertical')

const laneItems: KunTabItem[] = [
  { value: 'all', textValue: '全部', icon: 'lucide:layers' },
  { value: 'news', textValue: '情报', icon: 'lucide:newspaper' },
  { value: 'column', textValue: '专栏', icon: 'lucide:book-open-text' }
]

const sourceItems = computed<KunTabItem[]>(() => [
  { value: 'all', textValue: '全部来源', icon: 'lucide:globe' },
  ...props.sources.map((partner) => ({
    value: partner.key,
    textValue: partner.display_name,
    icon: 'lucide:rss'
  }))
])

const years = computed(() => props.archive?.years ?? [])
const months = computed(() => props.archive?.months ?? [])

// Collapsed by default outside the rail, where the year and month grids would
// otherwise push the first card most of a phone screen down. A link that
// arrives with a year already picked opens on it, so the reader can see what
// narrowed the list.
const isArchiveOpen = ref(props.orientation === 'vertical' || year.value > 0)
const isArchiveShown = computed(() => isRail.value || isArchiveOpen.value)

const range = computed(() => {
  if (!year.value) return ''
  return month.value ? `${year.value} 年 ${month.value} 月` : `${year.value} 年`
})

const selectYear = (value: number) => {
  year.value = value
  month.value = 0
}

const monthHref = computed(() => {
  const query = new URLSearchParams()
  if (lane.value !== 'all') query.set('lane', lane.value)
  if (source.value !== 'all') query.set('source', source.value)
  const search = query.toString()
  return `/news/${year.value}/${month.value}${search ? `?${search}` : ''}`
})
</script>

<template>
  <div :class="cn('space-y-5', !isRail && 'space-y-4')">
    <section class="space-y-2">
      <h2 v-if="isRail" class="text-default-500 px-1 text-xs font-medium">
        栏目
      </h2>
      <KunTab
        v-model="lane"
        :items="laneItems"
        variant="underlined"
        color="primary"
        :orientation="orientation"
        :full-width="isRail"
        :scrollable="!isRail"
      />
    </section>

    <section class="space-y-2">
      <h2 v-if="isRail" class="text-default-500 px-1 text-xs font-medium">
        来源
      </h2>
      <KunTab
        v-model="source"
        :items="sourceItems"
        variant="underlined"
        color="primary"
        :orientation="orientation"
        :full-width="isRail"
        :scrollable="!isRail"
      />
    </section>

    <section class="space-y-2">
      <h2 v-if="isRail" class="text-default-500 px-1 text-xs font-medium">
        归档
      </h2>
      <button
        v-else
        type="button"
        class="text-default-500 hover:text-default-700 flex items-center gap-1 px-1 text-xs font-medium transition-colors"
        @click="isArchiveOpen = !isArchiveOpen"
      >
        <span>归档</span>
        <span v-if="range" class="text-primary">{{ range }}</span>
        <KunIcon
          :name="isArchiveOpen ? 'lucide:chevron-up' : 'lucide:chevron-down'"
          class="size-3.5"
        />
      </button>

      <div v-if="isArchiveShown" class="flex flex-wrap gap-1.5">
        <KunButton
          size="sm"
          :variant="year ? 'light' : 'flat'"
          @click="selectYear(0)"
        >
          全部年份
        </KunButton>
        <KunButton
          v-for="entry in years"
          :key="entry.year"
          size="sm"
          :variant="year === entry.year ? 'flat' : 'light'"
          @click="selectYear(entry.year)"
        >
          {{ `${entry.year} 年` }}
          <span class="text-default-400 ml-1 text-xs tabular-nums">
            {{ entry.count }}
          </span>
        </KunButton>
      </div>

      <div
        v-if="year && isArchiveShown"
        :class="
          cn(
            'grid gap-1',
            isRail ? 'grid-cols-3' : 'grid-cols-4 sm:grid-cols-6'
          )
        "
      >
        <template v-if="months.length">
          <KunButton
            v-for="entry in months"
            :key="entry.month"
            size="sm"
            :variant="month === entry.month ? 'flat' : 'light'"
            :disabled="!entry.count"
            @click="month = month === entry.month ? 0 : entry.month"
          >
            {{ `${entry.month} 月` }}
            <span class="text-default-400 ml-1 text-xs tabular-nums">
              {{ entry.count }}
            </span>
          </KunButton>
        </template>
        <template v-else>
          <KunSkeleton v-for="n in 12" :key="n" height="2rem" rounded="lg" />
        </template>
      </div>

      <KunButton
        v-if="month && isArchiveShown"
        size="sm"
        variant="flat"
        color="primary"
        :full-width="isRail"
        :href="monthHref"
      >
        查看该月详情
      </KunButton>
    </section>
  </div>
</template>
