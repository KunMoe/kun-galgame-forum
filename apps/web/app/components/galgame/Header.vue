<script setup lang="ts">
import {
  GALGAME_RESOURCE_TYPE_ICON_MAP,
  GALGAME_RESOURCE_PLATFORM_ICON_MAP
} from '~/constants/galgameResource'
import {
  KUN_GALGAME_PLAY_STATE_MAP,
  type KunGalgamePlayState
} from '~/constants/galgame-playtime'
import {
  KUN_GALGAME_RESOURCE_TYPE_MAP,
  KUN_GALGAME_RESOURCE_LANGUAGE_MAP,
  KUN_GALGAME_RESOURCE_PLATFORM_MAP,
  KUN_GALGAME_CONTENT_LIMIT_MAP,
  KUN_USER_TEXT_CHIP_CLASS
} from '~/constants/galgame'
import { settle } from '#shared/utils/api/problem'

const props = defineProps<{
  galgame: GalgameDetail
}>()

const { isBlurred } = useContentStance()

const emits = defineEmits<{
  onRatingCreated: [GalgameRatingCardOnGalgamePage]
}>()

const { id } = usePersistUserStore()
const api = useApiClient()
const canBanResourcePublish = useCan('galgame.ban_resource_publish')

const resourcePublishBanned = inject<Ref<boolean>>(
  'galgameResourcePublishBanned',
  ref(false)
)
const banning = ref(false)
const toggleResourceBan = async () => {
  if (banning.value) {
    return
  }
  const willBan = !resourcePublishBanned.value
  const ok = await useComponentMessageStore().alert(
    willBan ? '禁止在本游戏下发布资源' : '解除资源发布禁止',
    willBan
      ? '部分游戏可能因为版权方通知，或者其余第三方原因导致不可用，此时需要禁止发布任何下载资源。'
      : '解除后，用户将可以重新在本游戏下发布下载资源。'
  )
  if (!ok) {
    return
  }
  banning.value = true
  const result = await settle(
    willBan
      ? api.PUT('/works/{work_id}/resource-publish-ban', {
          params: { path: { work_id: String(props.galgame.id) } }
        })
      : api.DELETE('/works/{work_id}/resource-publish-ban', {
          params: { path: { work_id: String(props.galgame.id) } }
        })
  )
  banning.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  resourcePublishBanned.value = result.data.is_resource_publish_banned
  useMessage(
    result.data.is_resource_publish_banned ? '已禁止发布资源' : '已解除禁止',
    'success'
  )
}

const galgameAliasArray = computed(() =>
  props.galgame.name_original
    ? [props.galgame.name_original, ...props.galgame.alias]
    : props.galgame.alias
)

const isRatingOpen = ref(false)
const ratingInvite = ref<KunGalgamePlayState | null>(null)

const onWantsRating = (state: KunGalgamePlayState) => {
  ratingInvite.value = state
}

const dismissRatingInvite = () => {
  ratingInvite.value = null
}

const openRatingFromInvite = () => {
  isRatingOpen.value = true
}

const onRatingPublished = (newRating: GalgameRatingCardOnGalgamePage) => {
  ratingInvite.value = null
  emits('onRatingCreated', newRating)
}

const isRatingDetailOpen = ref(false)
const ratingDetailSource = ref('')
const openRatingDetail = (source: string) => {
  ratingDetailSource.value = source
  isRatingDetailOpen.value = true
}

const coversOpen = ref(false)
const hasMoreCovers = computed(() => (props.galgame.covers?.length ?? 0) > 1)

const favoriteCount = ref(props.galgame.favorite_count)
const isFavorited = ref(props.galgame.is_favorited)

watch(
  () => props.galgame.favorite_count,
  (value) => (favoriteCount.value = value)
)
watch(
  () => props.galgame.is_favorited,
  (value) => (isFavorited.value = value)
)

const favoritePickerOpen = ref(false)
const { setFavorited } = useMyGalgameInteractions()

const openFavoritePicker = () => {
  if (!id) {
    useAuthModal().open()
    return
  }
  favoritePickerOpen.value = true
}

const onFavoriteSaved = (payload: { favorited: boolean }) => {
  if (isFavorited.value !== payload.favorited) {
    favoriteCount.value += payload.favorited ? 1 : -1
  }
  isFavorited.value = payload.favorited
  setFavorited(props.galgame.id, payload.favorited)
}
</script>

<template>
  <KunCard
    :is-hoverable="false"
    :is-transparent="false"
    content-class="grid grid-cols-[7rem_1fr] items-start gap-3 md:grid-cols-[220px_1fr]"
  >
    <div
      class="relative col-start-1 row-start-1 aspect-[5/7] w-full self-start overflow-hidden rounded-lg md:row-end-3"
    >
      <KunNsfwMask
        :active="isBlurred && galgame.content_limit !== 'sfw'"
        class-name="h-full"
        label="成人向封面已模糊"
      >
        <KunLightboxGallery>
          <KunLightboxGalleryItem
            :src="getEffectivePortrait(galgame)"
            :alt="galgame.name"
            :wrap="false"
            v-slot="{ open }"
          >
            <KunImage
              class="size-full cursor-zoom-in object-cover"
              :src="getEffectivePortrait(galgame)"
              loading="eager"
              fetchpriority="high"
              :thumbhash="resolvePortraitThumbhash(galgame)"
              :alt="galgame.name"
              @click="open"
            />
          </KunLightboxGalleryItem>
        </KunLightboxGallery>
      </KunNsfwMask>

      <div class="absolute top-2 right-2 z-10" @click.stop>
        <KunTooltip text="收藏">
          <span class="flex">
            <KunReaction
              :model-value="isFavorited"
              :toggle="false"
              size="sm"
              icon="lucide:heart"
              color="danger"
              label="收藏"
              class="bg-content1/90 py-1.5 shadow-sm backdrop-blur"
              :class="!isFavorited && 'text-default-700'"
              @click.stop="openFavoritePicker"
            />
          </span>
        </KunTooltip>
      </div>

      <KunChip
        variant="solid"
        class="absolute top-2 left-2"
        :color="galgame.content_limit === 'sfw' ? 'success' : 'danger'"
      >
        <KunTooltip
          position="right"
          :text="KUN_GALGAME_CONTENT_LIMIT_MAP[galgame.content_limit]"
        >
          {{ galgame.content_limit.toLocaleUpperCase() }}
        </KunTooltip>
      </KunChip>

      <button
        v-if="hasMoreCovers"
        type="button"
        class="bg-background/80 hover:bg-background shadow-kun-sm absolute right-2 bottom-2 z-10 inline-flex items-center gap-1.5 rounded-lg px-2 py-1.5 text-xs font-medium backdrop-blur transition-colors md:px-2.5"
        @click="coversOpen = true"
      >
        <KunIcon name="lucide:images" class="size-4" />
        <span class="hidden md:inline">查看所有封面</span>
      </button>
      <GalgameCovers
        v-model="coversOpen"
        :work-id="galgame.id"
        :covers="galgame.covers"
      />
    </div>

    <div class="col-start-2 row-start-1 flex min-w-0 flex-col gap-3">
      <div class="flex flex-wrap items-center gap-2">
        <h1 class="text-2xl md:text-3xl">
          {{ galgame.name }}
        </h1>
      </div>

      <KunScrollShadow
        axis="vertical"
        shadow-size="2rem"
        class-name="max-h-[100px]"
      >
        <div class="flex flex-wrap gap-2">
          <template v-for="(alias, index) in galgameAliasArray" :key="index">
            <KunChip v-if="alias" :class-name="KUN_USER_TEXT_CHIP_CLASS">{{
              alias
            }}</KunChip>
          </template>
        </div>
      </KunScrollShadow>
    </div>

    <div class="col-start-1 col-end-3 row-start-2 min-w-0 md:col-start-2">
      <div class="space-y-3">
        <KunDivider />

        <div class="space-y-1 space-x-1">
          <KunChip
            v-for="(t, index) in galgame.type"
            :key="index"
            color="primary"
          >
            <KunIcon :name="GALGAME_RESOURCE_TYPE_ICON_MAP[t]" />
            {{ KUN_GALGAME_RESOURCE_TYPE_MAP[t] }}
          </KunChip>

          <KunChip
            v-for="(lang, index) in galgame.language"
            :key="index"
            color="secondary"
          >
            <KunIcon class="icon" name="lucide:globe" />
            {{ KUN_GALGAME_RESOURCE_LANGUAGE_MAP[lang] }}
          </KunChip>

          <KunChip
            v-for="(platform, index) in galgame.platform"
            :key="index"
            color="success"
          >
            <KunIcon
              class="icon"
              :name="GALGAME_RESOURCE_PLATFORM_ICON_MAP[platform]"
            />
            {{ KUN_GALGAME_RESOURCE_PLATFORM_MAP[platform] }}
          </KunChip>
        </div>

        <GalgameHeaderRatingStrip
          :galgame="galgame"
          @open-rating="isRatingOpen = true"
          @open-detail="openRatingDetail"
        />

        <GalgameHeaderRatingModal
          v-model="isRatingDetailOpen"
          :galgame="galgame"
          :ratings="galgame.ratings"
          :source="ratingDetailSource"
        />

        <GalgameHeaderPlaytime
          :galgame="galgame"
          @wants-rating="onWantsRating"
        />

        <KunInfo
          v-if="ratingInvite"
          color="info"
          :title="`已标记为${KUN_GALGAME_PLAY_STATE_MAP[ratingInvite]}, 顺手给它打个分？`"
        >
          <div class="mt-2 flex items-center gap-2">
            <KunButton size="sm" color="primary" @click="openRatingFromInvite">
              去评分
            </KunButton>
            <KunButton
              size="sm"
              variant="light"
              color="default"
              @click="dismissRatingInvite"
            >
              不用了
            </KunButton>
          </div>
        </KunInfo>

        <div class="flex flex-wrap items-center gap-2">
          <div class="flex items-center gap-1">
            <KunReaction
              v-if="galgame.is_on_forum !== false"
              :count="galgame.view"
              :toggle="false"
              icon="lucide:eye"
              label="浏览量"
              disable-animation
              class="pointer-events-none"
            />

            <GalgameLike
              :work-id="galgame.id"
              :target-user-id="galgame.user.id"
              :like-count="galgame.like_count"
              :is-liked="galgame.is_liked"
            />

            <GalgameFavorite
              :work-id="galgame.id"
              :target-user-id="galgame.user.id"
              :favorite-count="favoriteCount"
              :is-favorited="isFavorited"
              @saved="onFavoriteSaved"
            />
          </div>

          <div class="ml-auto flex flex-wrap items-center justify-end gap-1">
            <KunButton
              variant="shadow"
              color="primary"
              size="sm"
              @click="isRatingOpen = true"
            >
              <span class="flex items-center gap-1">
                <KunIcon name="lucide:star" />添加评分
              </span>
            </KunButton>

            <GalgameDlsitePurchase
              v-if="galgame.dlsite_purchase_url"
              :purchase-url="galgame.dlsite_purchase_url"
              :coupon-url="galgame.dlsite_coupon_url"
              :campaign-name="galgame.dlsite_campaign_name"
            />

            <KunButton
              variant="light"
              color="default"
              size="sm"
              @click="navigateTo(`/galgame/${galgame.id}/edit`)"
            >
              <span class="flex items-center gap-1">
                <KunIcon name="lucide:file-pen-line" />编辑资料
              </span>
            </KunButton>

            <KunPopover
              v-if="galgame.user.id !== id || canBanResourcePublish"
              position="bottom-end"
            >
              <template #trigger>
                <KunButton
                  :is-icon-only="true"
                  variant="light"
                  color="default"
                  size="sm"
                >
                  <KunIcon name="lucide:ellipsis" />
                </KunButton>
              </template>
              <div class="flex w-44 flex-col gap-1 p-2">
                <ReportButton
                  v-if="galgame.user.id !== id"
                  menu
                  subject-kind="galgame"
                  :subject-id="galgame.id"
                  :snapshot="galgame.name"
                  :subject-url="`${kungal.domain.main}/galgame/${galgame.id}`"
                />
                <KunButton
                  v-if="canBanResourcePublish"
                  variant="light"
                  :color="resourcePublishBanned ? 'success' : 'danger'"
                  size="sm"
                  class-name="w-full justify-start gap-2"
                  :loading="banning"
                  @click="toggleResourceBan"
                >
                  <KunIcon
                    :name="
                      resourcePublishBanned
                        ? 'lucide:circle-check'
                        : 'lucide:ban'
                    "
                  />
                  {{
                    resourcePublishBanned ? '解除资源发布禁止' : '禁止发布资源'
                  }}
                </KunButton>
              </div>
            </KunPopover>

            <GalgameRatingPublish
              v-model="isRatingOpen"
              :work-id="galgame.id"
              :preset-play-state="ratingInvite ?? undefined"
              @on-published="onRatingPublished"
            />
          </div>
        </div>
      </div>
    </div>
  </KunCard>

  <GalgameCollectionPickerModal
    v-model="favoritePickerOpen"
    :work-id="galgame.id"
    @saved="onFavoriteSaved"
  />
</template>
