<script setup lang="ts">
import type { KunTabItem } from '@kungal/ui-vue'
import { useElementSize, useWindowSize } from '@vueuse/core'
import type { RatingPage, Work } from '#shared/utils/api/schemas'
import { ratingSummaryToPageCard } from '~/utils/galgame/ratingCard'
import { toKunUser } from '~/utils/userRef'

const props = defineProps<{
  galgame: Work
}>()

provide<Work>('galgame', props.galgame)

const resourcePublishBanned = ref(props.galgame.is_resource_publish_banned)
provide<Ref<boolean>>('galgameResourcePublishBanned', resourcePublishBanned)

const route = useRoute()
const router = useRouter()
const nameOf = useCatalogName()
const { id: currentUserId } = usePersistUserStore()
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
  // Only the quiz panel paginates, so ?page= belongs to that tab alone.
  delete query.page
  if (tab !== 'intro' && DEEP_LINK_TABS.includes(tab)) {
    query.tab = tab
  } else {
    delete query.tab
  }
  router.replace({ query })
})
const hasPatchResource = ref(false)

// `position: sticky; top: 5rem` on an element taller than the viewport freezes
// it at the top and never lets its bottom into view. Measured on an 820x1180
// tablet: the sidebar was 1631px and held at top=80 / bottom=1711 from scrollY
// 1000 all the way to 3400, so its last 531px only appeared once the 4109px
// content column had run out — 「往下滑动想看左侧栏目, 但是必须得等到右侧资源栏
// 划完」. It therefore sticks only while it fits. SSR has no measurement and
// renders it unstuck, which is the safe direction.
const sidebarRef = ref<HTMLElement | null>(null)
const { height: sidebarHeight } = useElementSize(sidebarRef)
const { height: viewportHeight } = useWindowSize()
const canStickSidebar = computed(
  () =>
    sidebarHeight.value > 0 &&
    sidebarHeight.value <= viewportHeight.value - 96
)

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

const { data: ratingsPage, refresh: refreshRatings } = await useApi<RatingPage>(
  () => `work-ratings:${props.galgame.id}`,
  (api, { signal }) =>
    api.GET('/ratings', {
      params: {
        query: {
          work_id: props.galgame.id,
          include_nsfw: true,
          limit: 100,
          sort: 'created_desc'
        }
      },
      signal
    })
)

const ratings = computed(() =>
  (ratingsPage.value?.items ?? []).map((item) =>
    ratingSummaryToPageCard(item, nameOf)
  )
)
const sortedRatings = computed(() => {
  return [...ratings.value].sort(
    (a, b) => b.short_summary.length - a.short_summary.length
  )
})

const handleRatingsChanged = () => {
  void refreshRatings()
}

const hasLocalRating = computed(() =>
  ratings.value.some((rating) => rating.user.id === currentUserId)
)

const creator = computed(() =>
  props.galgame.creator ? toKunUser(props.galgame.creator) : null
)
const hasCreator = computed(() => !!creator.value)
const hasContributorCard = computed(
  () => hasCreator.value || !!props.galgame.contributors.length
)

const workId = computed(() => Number(props.galgame.id))
</script>

<template>
  <div class="flex flex-col gap-3">
    <GalgameHeader
      :galgame="galgame"
      :ratings="ratings"
      :has-local-rating="hasLocalRating"
      @on-rating-created="handleRatingsChanged"
    />

    <div v-if="galgame.tags.length" class="md:hidden">
      <GalgameTag :tags="galgame.tags" variant="mobile" />
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
                <GalgameIntroduction :intros="galgame.intros" />

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

                <GalgameLink
                  :links="galgame.links"
                  :external-refs="galgame.external_refs"
                />
              </div>

              <GalgameStaff :credits="galgame.credits" />

              <GalgameCharacterPanel :roster="galgame.roster" />

              <GalgameGallery :screenshots="galgame.screenshots" />

              <GalgameSeriesPanel
                v-if="galgame.series.length"
                :series="galgame.series"
              />
            </KunTabPanel>

            <KunTabPanel value="resource" :loading="resourceLoading">
              <GalgameResource @update:loading="resourceLoading = $event" />
            </KunTabPanel>

            <KunTabPanel value="patch" :loading="patchLoading">
              <GalgamePatchContainer
                :work-id="workId"
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
        ref="sidebarRef"
        :class="
          cn(
            'order-2 flex min-w-0 flex-col gap-3 md:col-span-1 md:col-start-1 md:row-start-1 md:self-start',
            canStickSidebar && 'md:sticky md:top-20'
          )
        "
      >
        <div v-if="galgame.tags.length" class="hidden md:block">
          <GalgameTag :tags="galgame.tags" variant="desktop" />
        </div>

        <GalgameInfo
          :companies="galgame.companies"
          :engines="galgame.engines"
          :series="galgame.series"
          :content-rating="galgame.content_rating"
          :original-language="galgame.original_language"
          :release-date="galgame.release_date"
          :release-date-precision="galgame.release_date_precision"
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
            v-if="creator"
            class="text-default-500 flex cursor-default flex-wrap items-center gap-2"
          >
            <UserHoverCard :user-id="creator.id">
              <KunUserChip :user="creator" />
            </UserHoverCard>
            <span class="text-sm">
              <KunTime :time="galgame.created_at" type="date" show-year />
              创建本游戏
            </span>
          </div>

          <GalgameContributorContainer />
        </KunCard>
      </div>
    </div>
  </div>
</template>
