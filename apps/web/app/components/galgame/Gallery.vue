<script setup lang="ts">
import {
  galgameImageSourceLabel,
  galgameImageSourceRank
} from '~/constants/galgameImageSource'
import type { WorkScreenshot } from '#shared/utils/api/schemas'

const props = defineProps<{
  screenshots: WorkScreenshot[]
}>()

const { showKUNGalgameGallerySexualLevels: sexualLevels } = storeToRefs(
  usePersistSettingsStore()
)

const { allowsNsfw: showNsfw, isBlurred } = useContentStance()

const sexualLevel = (s: WorkScreenshot): number => {
  const sexual = s.image?.sexual
  if (sexual === 'suggestive') return 1
  if (sexual === 'explicit') return 2
  return 0
}

const sexualOk = (s: WorkScreenshot) =>
  showNsfw.value ||
  sexualLevel(s) === 0 ||
  sexualLevels.value.includes(sexualLevel(s))

// The two systems stack rather than replace each other: a level the reader
// ticked in the filter is an explicit choice and stays sharp, while one that
// is only here because the account allows NSFW at all is what 模糊 masks.
const isMasked = (s: WorkScreenshot) =>
  isBlurred.value &&
  sexualLevel(s) >= 1 &&
  !sexualLevels.value.includes(sexualLevel(s))

const allShots = computed(() =>
  [...(props.screenshots ?? [])].filter((s) => !!s.image)
)

const sourceKeys = computed(() =>
  [...new Set(allShots.value.map((s) => s.site))].sort(
    (a, b) => galgameImageSourceRank(a) - galgameImageSourceRank(b)
  )
)

const hiddenSources = ref<string[]>([])
const expandedSources = ref<string[]>([])
watch(sourceKeys, (keys) => {
  hiddenSources.value = hiddenSources.value.filter((k) => keys.includes(k))
  expandedSources.value = expandedSources.value.filter((k) => keys.includes(k))
})

const GROUP_PREVIEW = 8

const shotKey = (s: WorkScreenshot) => s.image?.hash ?? s.image?.url ?? ''

const groups = computed(() =>
  sourceKeys.value.map((key) => {
    const shots = allShots.value
      .filter((s) => s.site === key)
      .sort((a, b) => {
        if (a.sort_order !== b.sort_order) return a.sort_order - b.sort_order
        return shotKey(a).localeCompare(shotKey(b))
      })
    const shown = shots.filter((s) => sexualOk(s))
    const visible = expandedSources.value.includes(key)
      ? shown
      : shown.slice(0, GROUP_PREVIEW)
    const folded = shown.length - visible.length
    return {
      key,
      label: galgameImageSourceLabel(key),
      total: shots.length,
      shown,
      visible,
      folded,
      foldIndex: folded > 0 ? visible.length - 1 : -1,
      hidden: shots.length - shown.length
    }
  })
)

const expandSource = (key: string) => {
  expandedSources.value = [...expandedSources.value, key]
}

const openGroups = computed(() =>
  groups.value.filter((g) => !hiddenSources.value.includes(g.key))
)
const shownCount = computed(() =>
  openGroups.value.reduce((n, g) => n + g.shown.length, 0)
)
const hiddenCount = computed(() => allShots.value.length - shownCount.value)

const showHeaders = computed(() => sourceKeys.value.length > 1)

const sexualCounts = computed(() => {
  const counts: Record<number, number> = { 1: 0, 2: 0, 3: 0 }
  for (const s of allShots.value) {
    const level = sexualLevel(s)
    if (level >= 1 && level <= 3) counts[level] = (counts[level] ?? 0) + 1
  }
  return counts
})

const hasRated = computed(() => allShots.value.some((s) => sexualLevel(s) >= 1))
const canFilter = computed(() => hasRated.value || showHeaders.value)

const sourceOptions = computed(() =>
  groups.value.map((g) => ({
    key: g.key,
    label: g.label,
    total: g.total,
    on: !hiddenSources.value.includes(g.key)
  }))
)

const toggleSource = (key: string) => {
  hiddenSources.value = hiddenSources.value.includes(key)
    ? hiddenSources.value.filter((k) => k !== key)
    : [...hiddenSources.value, key]
}

const thumbSrc = (s: WorkScreenshot) =>
  s.image?.url ? withImageVariant(s.image.url, 'mini') : ''

const shotSrc = (s: WorkScreenshot) => s.image?.url ?? ''

const RING_W = 2.5
const RING_DEPTH: Record<number, number> = { 1: 60, 2: 80, 3: 100 }
const ringColor = (token: 'warning', level: number) =>
  `color-mix(in oklab, var(--color-${token}) ${RING_DEPTH[level] ?? 100}%, transparent)`

const ratingRing = (s: WorkScreenshot) => {
  const level = sexualLevel(s)
  if (level < 1) return {}
  return {
    boxShadow: `inset 0 0 0 ${RING_W}px ${ringColor('warning', level)}`
  }
}
</script>

<template>
  <div v-if="allShots.length" class="space-y-3">
    <div class="flex flex-wrap items-end justify-between gap-2">
      <KunHeader
        name="画廊"
        description="该 Galgame 的截图 / CG 集"
        scale="h3"
      />
      <GalgameGalleryFilter
        v-if="canFilter"
        :show-nsfw="showNsfw"
        :hidden-count="hiddenCount"
        :sexual-counts="sexualCounts"
        :sources="sourceOptions"
        @toggle-source="toggleSource"
      />
    </div>

    <KunLightboxGallery v-if="shownCount">
      <div class="space-y-5">
        <section v-for="g in openGroups" :key="g.key" class="space-y-2">
          <div v-if="showHeaders" class="flex flex-wrap items-center gap-2">
            <h3 class="text-default-600 text-sm font-medium">
              {{ g.label }}
              <span class="text-default-400">({{ g.total }})</span>
            </h3>
            <span v-if="g.hidden" class="text-default-400 text-xs">
              已隐藏 {{ g.hidden }} 张
            </span>
          </div>

          <p
            v-if="!g.shown.length"
            class="text-default-400 border-default/20 rounded-lg border border-dashed px-3 py-4 text-xs"
          >
            {{ g.total }} 张图片已按分级隐藏,点击「筛选」调整
          </p>

          <div
            v-else
            class="grid grid-cols-2 gap-2 sm:grid-cols-[repeat(auto-fill,minmax(180px,1fr))]"
          >
            <KunLightboxGalleryItem
              v-for="(s, i) in g.visible"
              :key="shotKey(s)"
              :src="shotSrc(s)"
              :alt="s.caption || ''"
              :wrap="false"
              v-slot="{ open }"
            >
              <button
                type="button"
                class="group hover:ring-primary focus:ring-primary relative block w-full overflow-hidden rounded-lg ring-1 ring-transparent transition-all focus:outline-none"
                :aria-label="
                  i === g.foldIndex ? '显示全部截图' : s.caption || '查看截图'
                "
                @click="i === g.foldIndex ? expandSource(g.key) : open()"
              >
                <KunNsfwMask :active="isMasked(s)" label="成人向截图已模糊">
                  <KunImage
                    :src="thumbSrc(s)"
                    :alt="s.caption || ''"
                    loading="lazy"
                    object-fit="cover"
                    :thumbhash="s.image?.thumbhash ?? undefined"
                    class="h-full w-full cursor-zoom-in object-cover transition-transform duration-200 group-hover:scale-105"
                    :style="{ aspectRatio: '16/9' }"
                  />
                </KunNsfwMask>
                <div
                  v-if="s.caption"
                  class="absolute right-0 bottom-0 left-0 truncate bg-black/50 px-2 py-1 text-xs text-white opacity-0 transition-opacity group-hover:opacity-100"
                >
                  {{ s.caption }}
                </div>
                <div
                  v-if="sexualLevel(s) >= 1"
                  class="pointer-events-none absolute inset-0 rounded-lg"
                  :style="ratingRing(s)"
                />
                <div
                  v-if="i === g.foldIndex"
                  class="absolute inset-0 flex flex-col items-center justify-center rounded-lg bg-black/60 text-white"
                >
                  <span class="text-lg font-medium">+{{ g.folded }}</span>
                  <span class="text-xs">显示全部</span>
                </div>
              </button>
            </KunLightboxGalleryItem>
          </div>
        </section>
      </div>
    </KunLightboxGallery>

    <KunNull
      v-else
      :description="`${hiddenCount} 张图片已按分级 / 来源隐藏,点击「筛选」调整`"
    />
  </div>
</template>
