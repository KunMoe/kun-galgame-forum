<script setup lang="ts">
import type { KunTabItem } from '@kungal/ui-vue'

const props = defineProps<{
  galgame: GalgameDetail
}>()

provide<GalgameDetail>('galgame', props.galgame)

const resourcePublishBanned = ref(props.galgame.resource_publish_banned)
provide<Ref<boolean>>('galgameResourcePublishBanned', resourcePublishBanned)

const route = useRoute()
const router = useRouter()
const DEEP_LINK_TABS = ['intro', 'resource', 'comment', 'quiz']
const initialTab = () => {
  if (route.query.comment) {
    return 'comment'
  }
  const tab = route.query.tab
  return typeof tab === 'string' && DEEP_LINK_TABS.includes(tab) ? tab : 'intro'
}
const activeTab = ref(initialTab())

watch(activeTab, (tab) => {
  const query = { ...route.query }
  delete query.comment
  delete query.thread
  if (tab !== 'intro' && DEEP_LINK_TABS.includes(tab)) {
    query.tab = tab
  } else {
    delete query.tab
  }
  router.replace({ query })
})
const hasPatchResource = ref(false)

const resourceLoading = ref(false)
const patchLoading = ref(false)
const commentLoading = ref(false)
const quizLoading = ref(false)

const contentTabs = computed<KunTabItem[]>(() => [
  { value: 'intro', textValue: '游戏介绍', icon: 'lucide:book-open' },
  { value: 'resource', textValue: '本体资源下载', icon: 'lucide:download' },
  ...(hasPatchResource.value
    ? [{ value: 'patch', textValue: '补丁资源下载', icon: 'lucide:puzzle' }]
    : []),
  { value: 'comment', textValue: '评论区', icon: 'lucide:messages-square' },
  { value: 'quiz', textValue: '题库', icon: 'lucide:brain' }
])

const ratings = ref([...props.galgame.ratings])
const sortedRatings = computed(() => {
  return [...ratings.value].sort(
    (a, b) => b.short_summary.length - a.short_summary.length
  )
})

const handleRatingCreated = (newRating: GalgameRatingCardOnGalgamePage) => {
  ratings.value.unshift(newRating)
}

const hasCreator = computed(() => props.galgame.user.id > 0)
const hasContributorCard = computed(
  () => hasCreator.value || !!props.galgame.contributor?.length
)
</script>

<template>
  <div class="flex flex-col gap-3">
    <GalgameHeader
      :galgame="galgame"
      @on-rating-created="handleRatingCreated"
    />

    <div v-if="galgame.tag?.length" class="md:hidden">
      <GalgameTag :tags="galgame.tag" variant="mobile" />
    </div>

    <div
      v-if="sortedRatings.length && sortedRatings.length >= 3"
      class="grid grid-cols-1 gap-3"
    >
      <GalgameRatingRadarCard :ratings="sortedRatings" />
    </div>

    <div class="grid grid-cols-1 gap-3 md:grid-cols-3">
      <!-- Explicit md placement, NOT auto-placement + md:order-*. The sidebar
           is last in source order and this page's SSR HTML is ~1MB, so on a
           slow link the browser lays this column out while the sidebar element
           does not exist yet: as the only grid item it auto-placed into
           columns 1-2, and the whole page jumped ~430px right the moment the
           sidebar finished streaming in.
           row-start-1 is not decoration. `order` still reverses these two, so
           auto-placement walks the cursor to column 3 for this item and then
           has to go BACKWARDS to column 1 for the sidebar — which per the grid
           spec bumps it to a new row. Without it the sidebar rendered below
           the whole page instead of beside it. -->
      <div
        class="order-1 flex min-w-0 flex-col gap-3 md:col-span-2 md:col-start-2 md:row-start-1"
      >
        <KunTab
          v-model="activeTab"
          :items="contentTabs"
          variant="solid"
          size="md"
          inner-class-name="bg-[oklch(var(--content1))]!"
        />

        <KunCard
          :is-hoverable="false"
          :is-transparent="false"
          content-class="relative"
        >
          <KunTabPanels v-model="activeTab">
            <KunTabPanel value="intro" class-name="space-y-12">
              <div class="space-y-3">
                <GalgameIntroduction :introduction="galgame.introduction" />

                <div
                  v-if="sortedRatings.length && sortedRatings.length < 3"
                  class="space-y-1"
                >
                  <GalgameRatingRow
                    v-for="rating in sortedRatings"
                    :key="rating.id"
                    :rating="rating"
                  />
                </div>

                <GalgameLink :refs="galgame.refs" />
              </div>

              <GalgameStaff :staff="galgame.staff" />

              <GalgameCharacterPanel :characters="galgame.characters ?? []" />

              <GalgameGallery :screenshots="galgame.screenshots" />

              <GalgameSeriesPanel
                v-if="galgame.series?.length"
                :series="galgame.series"
              />
            </KunTabPanel>

            <KunTabPanel value="resource" :loading="resourceLoading">
              <GalgameResource @update:loading="resourceLoading = $event" />
            </KunTabPanel>

            <KunTabPanel
              v-if="galgame.vndb_id"
              value="patch"
              :loading="patchLoading"
            >
              <GalgamePatchContainer
                :vndb-id="galgame.vndb_id"
                @has-resource="hasPatchResource = $event"
                @update:loading="patchLoading = $event"
              />
            </KunTabPanel>

            <KunTabPanel value="comment" :loading="commentLoading">
              <GalgameCommentCommunityContainer
                @update:loading="commentLoading = $event"
              />
            </KunTabPanel>

            <KunTabPanel value="quiz" :loading="quizLoading">
              <GalgameQuizGalgamePanel @update:loading="quizLoading = $event" />
            </KunTabPanel>
          </KunTabPanels>
        </KunCard>
      </div>

      <div
        class="order-2 flex min-w-0 flex-col gap-3 md:sticky md:top-20 md:col-span-1 md:col-start-1 md:row-start-1 md:self-start"
      >
        <div v-if="galgame.tag?.length" class="hidden md:block">
          <GalgameTag :tags="galgame.tag" variant="desktop" />
        </div>

        <GalgameInfo
          :official="galgame.official"
          :engine="galgame.engine"
          :series="galgame.series"
          :age-limit="galgame.age_limit"
          :original-language="galgame.original_language"
          :release-date="galgame.release_date"
          :release-date-tba="galgame.release_date_tba"
        />

        <KunCard
          v-if="hasContributorCard"
          content-class="space-y-3"
          :is-hoverable="false"
          :is-transparent="false"
        >
          <KunHeader
            name="贡献者"
            description="本游戏项目的贡献者, 计 Galgame 资源发布贡献"
            scale="h3"
          />

          <div
            v-if="hasCreator"
            class="text-default-500 flex cursor-default flex-wrap items-center gap-2"
          >
            <KunUserChip :user="galgame.user" />
            <span class="text-sm">
              <KunTime :time="galgame.created" type="date" show-year />
              创建本游戏
            </span>
          </div>

          <GalgameContributorContainer />
        </KunCard>
      </div>
    </div>
  </div>
</template>
