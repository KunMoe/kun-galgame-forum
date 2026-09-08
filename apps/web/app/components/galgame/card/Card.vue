<script setup lang="ts" generic="T extends GalgameCard">
import { storeToRefs } from 'pinia'
import { GALGAME_RESOURCE_PLATFORM_ICON_MAP } from '~/constants/galgameResource'
import {
  KUN_GALGAME_LOCAL_RATING_META,
  kunGalgameRatingTierBadge
} from '~/constants/galgame-rating'

const props = defineProps<{
  galgames: T[]
  isTransparent?: boolean
  // A company's own page otherwise repeats its name under all 31 of its
  // games; works listed there through an imprint keep theirs.
  hideCompany?: string
}>()

defineSlots<{
  meta?: (props: { galgame: T }) => unknown
}>()

// The column count below is a CONTAINER query, but the reader's choice is about
// their phone, so it is gated on the viewport instead: on a phone this grid is
// the page's whole width, and the calendar's copy of it — the reason the rest
// of this cascade is container-based — has already stacked to full width by
// then. Above sm the container rules take over untouched.
const { showKUNGalgamePhoneColumns } = storeToRefs(usePersistSettingsStore())

const {
  showPlatform,
  showRating,
  showViewLike,
  showUpdateTime,
  showNsfwBadge,
  showCompany,
  showJapaneseName,
  isOpenInNewTab
} = storeToRefs(usePersistGalgameCardStore())

const ratingOf = (galgame: T) => {
  if (!galgame.rating || !galgame.rating_count) {
    return null
  }
  return {
    score: galgame.rating.toFixed(1),
    color:
      kunGalgameRatingTierBadge(
        KUN_GALGAME_LOCAL_RATING_META,
        galgame.rating,
        galgame.rating_count
      )?.color ?? 'default'
  }
}

const cards = computed(() =>
  props.galgames.map((galgame) => {
    const company =
      showCompany.value && galgame.company !== props.hideCompany
        ? galgame.company
        : ''
    const rating = showRating.value ? ratingOf(galgame) : null
    return {
      galgame,
      // No `variant: 'mini'` here: the image service's only variant is a fixed
      // 460x259 16:9 crop of whatever the source shape was, and cropping that
      // band again into the 5/7 poster box rendered a zoomed-in sliver.
      cover: getEffectivePortrait(galgame),
      thumbhash: resolvePortraitThumbhash(galgame),
      href: galgame.id > 0 ? `/galgame/${galgame.id}` : undefined,
      updateTime: showUpdateTime.value ? galgame.resource_update_time : '',
      isSfw: galgame.content_limit === 'sfw',
      company,
      rating,
      hasFooter: !!company || !!rating
    }
  })
)
</script>

<template>
  <!-- Container queries, not viewport ones: the calendar hangs this same grid
       in a column beside the day tiles, and a viewport-sized `xl:grid-cols-6`
       drew six ~115px thumbnails in it. -->
  <div class="@container">
    <div
      :class="
        cn(
          'grid grid-cols-2 gap-2 @lg:grid-cols-3 @lg:gap-3 @2xl:grid-cols-4 @4xl:grid-cols-5 @5xl:grid-cols-6',
          showKUNGalgamePhoneColumns === 3 && 'max-sm:grid-cols-3 max-sm:gap-1.5'
        )
      "
    >
      <KunCard
        :is-transparent="isTransparent"
        v-for="card in cards"
        :key="card.galgame.catalog_id ?? card.galgame.id"
        :href="card.href"
        :target="isOpenInNewTab ? '_blank' : undefined"
        class-name="p-0"
      >
        <div class="relative overflow-hidden">
          <KunImage
            v-if="card.cover"
            :src="card.cover"
            loading="lazy"
            :alt="card.galgame.name"
            :thumbhash="card.thumbhash"
            aspect-ratio="5 / 7"
          />
          <div
            v-else
            class="bg-default-100 text-default-400 flex items-center justify-center"
            style="aspect-ratio: 5 / 7"
          >
            <KunIcon name="lucide:image-off" class="size-6" />
          </div>

          <div
            v-if="showPlatform && card.galgame.platform.length"
            class="absolute top-2 left-2 flex flex-wrap gap-1"
            :class="showNsfwBadge ? 'right-7' : 'right-2'"
          >
            <span
              v-for="(platform, i) in card.galgame.platform"
              :key="i"
              class="bg-background flex size-6 items-center justify-center rounded-full p-1.5 text-xs backdrop-blur-sm"
            >
              <KunIcon
                :name="GALGAME_RESOURCE_PLATFORM_ICON_MAP[platform]"
                class="h-4 w-4"
              />
            </span>
          </div>

          <div
            v-if="showNsfwBadge"
            class="absolute top-0 right-0 size-5 [clip-path:polygon(100%_0,100%_100%,0_0)]"
            :class="card.isSfw ? 'bg-success' : 'bg-danger'"
            :title="card.galgame.content_limit.toLocaleUpperCase()"
          />

          <!-- SANCTIONED EXCEPTION to 铁律 #1 (no gradients): a bottom-to-top
             black scrim so the caption stays legible over an arbitrary cover.
             Listed in CLAUDE.md; do NOT remove it in a no-gradient sweep. -->
          <div
            v-if="
              (showViewLike || card.updateTime) &&
              card.galgame.is_on_forum !== false
            "
            class="absolute right-0 bottom-0 left-0 flex items-center gap-2 bg-gradient-to-t from-black/60 to-transparent p-2 text-xs transition-opacity duration-300"
          >
            <div v-if="showViewLike" class="flex shrink-0 gap-3">
              <span class="flex items-center gap-1">
                <KunIcon class="text-white" name="lucide:eye" />
                <span class="text-white">{{ card.galgame.view }}</span>
              </span>

              <span class="flex items-center gap-1">
                <KunIcon class="text-white" name="lucide:thumbs-up" />
                <span class="text-white">{{ card.galgame.like_count }}</span>
              </span>
            </div>

            <KunTime
              v-if="card.updateTime"
              class="ml-auto shrink-0 text-white!"
              :time="card.updateTime"
            />
          </div>
        </div>

        <div class="flex flex-auto flex-col p-2">
          <h2
            class="hover:text-primary line-clamp-2 text-sm font-medium transition-colors"
          >
            {{ card.galgame.name }}
          </h2>

          <p
            v-if="showJapaneseName && card.galgame.name_original"
            class="text-default-500 mt-1 line-clamp-1 text-xs"
          >
            {{ card.galgame.name_original }}
          </p>

          <slot name="meta" :galgame="card.galgame" />

          <div
            v-if="card.hasFooter"
            class="mt-auto flex min-w-0 items-center gap-1.5 pt-2"
          >
            <span
              v-if="card.company"
              class="text-default-500 truncate text-xs"
              :title="card.company"
            >
              {{ card.company }}
            </span>

            <KunChip
              v-if="card.rating"
              class-name="ml-auto shrink-0 tabular-nums"
              size="xs"
              variant="flat"
              :color="card.rating.color"
            >
              <template #start>
                <KunIcon name="lucide:star" class="size-3" />
              </template>
              {{ card.rating.score }}
            </KunChip>
          </div>
        </div>
      </KunCard>
    </div>
  </div>
</template>
