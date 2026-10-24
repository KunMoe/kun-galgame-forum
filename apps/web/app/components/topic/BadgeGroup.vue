<script setup lang="ts">
import { topicMiniAppsOf } from './miniapp/registry'

const props = withDefaults(
  defineProps<{
    section?: string
    upvoteTime?: Date | string | null
    hasBestAnswer?: boolean
    miniApps?: string[]
    isNSFWTopic?: boolean
    isNavToSection?: boolean
  }>(),
  {
    section: undefined,
    upvoteTime: null,
    miniApps: () => [],
    isNavToSection: false
  }
)

const miniApps = computed(() => topicMiniAppsOf(props.miniApps))

const { sectionOf } = useTopicSections()
const sectionInfo = computed(() => sectionOf(props.section))

const isRecentlyUpvoted = useState(`kun-recent-upvote-${useId()}`, () =>
  hourDiff(props.upvoteTime || 0, 24)
)
onMounted(() => {
  isRecentlyUpvoted.value = hourDiff(props.upvoteTime || 0, 24)
})

const handleClickSection = async () => {
  if (props.isNavToSection && sectionInfo.value) {
    await navigateTo(`/section/${sectionInfo.value.section}`)
  }
}
</script>

<template>
  <div class="flex flex-wrap items-center gap-2">
    <KunChip
      variant="solid"
      color="warning"
      v-if="upvoteTime && isRecentlyUpvoted"
    >
      <KunIcon name="lucide:sparkles" class="size-4 text-inherit" />
      <span class="text-inherit">该话题被推</span>
    </KunChip>

    <span v-if="hasBestAnswer" class="flex gap-1">
      <KunChip variant="solid" color="success">
        <KunIcon name="lucide:bookmark-check" class="size-4 text-inherit" />
        有解答
      </KunChip>
    </span>

    <span v-for="app in miniApps" :key="app.key" class="flex gap-1">
      <KunChip variant="solid" :color="app.badgeColor">
        <KunIcon :name="app.icon" class="size-4 text-inherit" />
        {{ app.badge }}
      </KunChip>
    </span>

    <span v-if="isNSFWTopic" class="flex gap-1">
      <KunChip variant="solid" color="primary" class-name="bg-orange-600">
        <KunIcon name="uil:18-plus" class="size-4 text-inherit" />
        NSFW 话题
      </KunChip>
    </span>

    <span v-if="sectionInfo" class="flex gap-1">
      <KunChip
        :color="sectionInfo.color"
        @click="handleClickSection"
        :class-name="cn(props.isNavToSection ? 'cursor-pointer' : '')"
      >
        <KunIcon :name="sectionInfo.icon_name" class="size-4 text-inherit" />
        {{ sectionInfo.label }}
      </KunChip>
    </span>
  </div>
</template>
