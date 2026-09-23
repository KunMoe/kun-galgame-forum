<script setup lang="ts">
import {
  KUN_GALGAME_TAG_CATEGORY_MAP,
  KUN_GALGAME_TAG_SPOILER_MAP,
  type KunGalgameTagSpoiler
} from '~/constants/galgameTag'
import type { WorkTag } from '#shared/utils/api/schemas'
import { catalogVocabularyName } from '#shared/utils/catalogName'

const props = withDefaults(
  defineProps<{
    tags: WorkTag[]
    variant?: 'mobile' | 'desktop'
  }>(),
  { variant: 'desktop' }
)

const isMobile = computed(() => props.variant === 'mobile')

const { allowsNsfw: isNsfwEnabled } = useContentStance()

const selectedCategories = ref<string[]>(
  isNsfwEnabled.value
    ? ['content', 'meta', 'technical', 'sexual']
    : ['content', 'meta', 'technical']
)

// The server drops every sexual tag from the detail payload for an SFW reader,
// so the 成人内容 box had nothing to match no matter how it was ticked.
const isCategoryLocked = (category: string) =>
  category === 'sexual' && !isNsfwEnabled.value

const selectedSpoilerLevels = ref<KunGalgameTagSpoiler[]>([0])

const toggleItemInArray = <T,>(arrayRef: Ref<T[]>, item: T) => {
  const index = arrayRef.value.indexOf(item)
  if (index === -1) {
    arrayRef.value.push(item)
  } else {
    arrayRef.value.splice(index, 1)
  }
}

const toggleCategory = (category: string) => {
  toggleItemInArray(selectedCategories, category)
}

const toggleSpoilerLevel = (spoiler: KunGalgameTagSpoiler) => {
  toggleItemInArray(selectedSpoilerLevels, spoiler)
}

const categoryOf = (tag: WorkTag) =>
  tag.is_sexual ? 'sexual' : tag.tag_kind

const spoilerOf = (tag: WorkTag): number =>
  tag.spoiler === 'major' ? 2 : tag.spoiler === 'minor' ? 1 : 0

const spoilerCounts = computed(() => {
  const counts: Record<number, number> = { 0: 0, 1: 0, 2: 0 }
  for (const tag of props.tags) {
    if (selectedCategories.value.includes(categoryOf(tag))) {
      counts[spoilerOf(tag)] = (counts[spoilerOf(tag)] ?? 0) + 1
    }
  }
  return counts
})

const filteredTags = computed(() => {
  if (
    selectedCategories.value.length === 0 ||
    selectedSpoilerLevels.value.length === 0
  ) {
    return []
  }

  const filtered = props.tags.filter(
    (tag) =>
      selectedCategories.value.includes(categoryOf(tag)) &&
      selectedSpoilerLevels.value.includes(spoilerOf(tag) as 0)
  )
  return filtered.sort((a, b) => Number(a.id) - Number(b.id))
})

const countColorByCategory = (category: string): string => {
  if (category === 'content') return 'text-primary'
  if (category === 'sexual') return 'text-danger'
  if (category === 'meta' || category === 'technical') return 'text-success'
  return 'text-default-500'
}
</script>

<template>
  <KunCard
    :is-hoverable="false"
    :is-transparent="false"
    class-name="overflow-visible"
    content-class="space-y-3"
  >
    <KunScrollShadow v-if="isMobile" shadow-size="5rem">
      <div class="flex w-fit items-center gap-3 whitespace-nowrap">
        <KunCheckBox
          v-for="(name, key) in KUN_GALGAME_TAG_CATEGORY_MAP"
          :key="key"
          class-name="gap-2"
          :model-value="selectedCategories.includes(key)"
          :disabled="isCategoryLocked(key)"
          color="primary"
          @click="toggleCategory(key)"
        >
          {{ name }}
        </KunCheckBox>

        <KunCheckBox
          v-for="(name, key) in KUN_GALGAME_TAG_SPOILER_MAP"
          :key="key"
          class-name="gap-2"
          :model-value="selectedSpoilerLevels.includes(Number(key) as 0)"
          color="primary"
          @click="toggleSpoilerLevel(Number(key) as KunGalgameTagSpoiler)"
        >
          {{ name }}
          <span class="text-default-400 text-xs">
            {{ spoilerCounts[Number(key)] }}
          </span>
        </KunCheckBox>
      </div>
    </KunScrollShadow>

    <KunScrollShadow
      axis="vertical"
      shadow-size="3rem"
      :class-name="
        isMobile ? 'max-h-[300px]' : 'max-h-[200px] md:max-h-[400px]'
      "
    >
      <TransitionGroup name="tag-list" tag="div" class="flex flex-wrap gap-1.5">
        <KunLink
          v-for="tag in filteredTags"
          :key="tag.id"
          underline="none"
          :to="`/galgame/tag/${tag.id}`"
        >
          <KunChip
            class-name="bg-default-500/10 cursor-pointer"
            :size="isMobile ? 'md' : 'sm'"
          >
            {{ catalogVocabularyName(tag) }}
            <span
              v-if="tag.catalog_work_count > 0"
              :class="cn('text-xs', countColorByCategory(categoryOf(tag)))"
            >
              {{ `+${tag.catalog_work_count}` }}
            </span>
            <span v-if="spoilerOf(tag) > 0" class="text-warning-600 text-xs">
              {{ spoilerOf(tag) > 1 ? '(严重剧透)' : '(剧透)' }}
            </span>
          </KunChip>
        </KunLink>
      </TransitionGroup>

      <KunNull
        v-if="filteredTags.length === 0"
        description="请至少选择一个类别来查看标签，或调整剧透等级"
      />
    </KunScrollShadow>

    <p v-if="!isNsfwEnabled" class="text-default-400 text-xs">
      成人内容标签未加载, 在设置面板打开 NSFW 开关后可见
    </p>

    <KunPopover v-if="!isMobile" position="top-start" full-width>
      <template #trigger>
        <KunButton variant="flat" color="primary" size="sm" full-width>
          <KunIcon name="lucide:filter" />
          筛选标签
        </KunButton>
      </template>

      <div class="min-w-[240px] space-y-4 p-4">
        <div class="space-y-2">
          <p class="text-default-500 text-xs font-medium">标签类型</p>
          <div class="flex flex-wrap gap-3">
            <KunCheckBox
              v-for="(name, key) in KUN_GALGAME_TAG_CATEGORY_MAP"
              :key="key"
              class-name="gap-2"
              :model-value="selectedCategories.includes(key)"
              :disabled="isCategoryLocked(key)"
              color="primary"
              @click="toggleCategory(key)"
            >
              {{ name }}
            </KunCheckBox>
          </div>
          <p v-if="!isNsfwEnabled" class="text-default-400 text-xs">
            成人内容标签未加载, 在设置面板打开 NSFW 开关后可见
          </p>
        </div>

        <div class="space-y-2">
          <p class="text-default-500 text-xs font-medium">剧透等级</p>
          <div class="flex flex-wrap gap-3">
            <KunCheckBox
              v-for="(name, key) in KUN_GALGAME_TAG_SPOILER_MAP"
              :key="key"
              class-name="gap-2"
              :model-value="selectedSpoilerLevels.includes(Number(key) as 0)"
              color="primary"
              @click="toggleSpoilerLevel(Number(key) as KunGalgameTagSpoiler)"
            >
              {{ name }}
              <span class="text-default-400 text-xs">
                {{ spoilerCounts[Number(key)] }}
              </span>
            </KunCheckBox>
          </div>
        </div>
      </div>
    </KunPopover>
  </KunCard>
</template>

<style scoped>
.tag-list-move,
.tag-list-enter-active,
.tag-list-leave-active {
  transition: all 0.5s cubic-bezier(0.55, 0, 0.1, 1);
}
.tag-list-enter-from,
.tag-list-leave-to {
  opacity: 0;
  transform: scale(0.8);
}
.tag-list-leave-active {
  position: absolute;
}
</style>
