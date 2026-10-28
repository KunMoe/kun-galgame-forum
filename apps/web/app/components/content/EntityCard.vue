<script setup lang="ts">
import type { EntityTarget } from '#shared/utils/api/schemas'
import { entityCardModel } from '~/utils/entityCardModel'
import type { EntityRefType } from '~/utils/entityUrl'

const props = defineProps<{
  target: EntityTarget
  inert?: boolean
}>()

const { showKUNGalgamePreferOriginalName } = storeToRefs(
  usePersistSettingsStore()
)
const { stance } = useContentStance()
const { open } = useEntityPreviewPanel()

const card = computed(() =>
  entityCardModel(
    props.target,
    showKUNGalgamePreferOriginalName.value,
    stance.value
  )
)

const ACCENT: Record<EntityRefType, string> = {
  work: 'text-primary-600',
  series: 'text-primary-600',
  character: 'text-secondary-600',
  credit_name: 'text-secondary-600',
  company: 'text-success-600',
  galgame_resource: 'text-success-600',
  tag: 'text-warning-600',
  rating: 'text-warning-600',
  topic: 'text-primary-600',
  user: 'text-default-600'
}

const BADGE_TONE = {
  danger: 'bg-danger-100 text-danger-600',
  warning: 'bg-warning-100 text-warning-700',
  primary: 'bg-primary-100 text-primary-600'
} as const

const PILL =
  'border-default-200 bg-content1 flex max-w-full items-center gap-2 rounded-full border px-3 py-1.5 text-sm'

const preview = (e: MouseEvent) => {
  if (
    props.inert ||
    card.value.isUnavailable ||
    e.button !== 0 ||
    e.metaKey ||
    e.ctrlKey ||
    e.shiftKey ||
    e.altKey
  ) {
    return
  }
  e.preventDefault()
  open(props.target)
}
</script>

<template>
  <!-- The click stops here so the article's lightbox, which claims every img
       inside it, never sees a click on this card's cover. It is taken in the
       capture phase so NuxtLink sees it prevented and does not navigate. -->
  <div
    :class="
      cn(
        'kun-content-entity',
        !inert && 'my-4',
        card.layout === 'pill' && 'w-fit max-w-full'
      )
    "
    @click.capture="preview"
    @click="!inert && $event.stopPropagation()"
  >
    <template v-if="card.layout === 'pill'">
      <span v-if="inert" :class="PILL">
        <ContentEntityPill
          :card="card"
          :accent="ACCENT[card.kind]"
          :badge-tone="BADGE_TONE"
        />
      </span>
      <KunLink
        v-else
        :to="card.href"
        underline="none"
        color="default"
        :class-name="cn(PILL, 'hover:bg-default-100 transition-colors')"
      >
        <ContentEntityPill
          :card="card"
          :accent="ACCENT[card.kind]"
          :badge-tone="BADGE_TONE"
        />
      </KunLink>
    </template>

    <KunCard
      v-else
      :href="inert ? undefined : card.href"
      :is-hoverable="!inert"
      bordered
      padding="none"
      :class-name="
        cn(card.layout === 'full' && 'h-36', card.isUnavailable && 'opacity-70')
      "
      :content-class="
        cn(
          'flex h-full flex-row items-center justify-start p-3',
          card.layout === 'full' ? 'gap-4' : 'gap-3'
        )
      "
    >
      <ContentEntityCardMedia
        :media="card.media"
        :accent="ACCENT[card.kind]"
        :compact="card.layout === 'compact'"
        :top-anchored="card.kind === 'character'"
      />

      <div class="flex h-full min-w-0 flex-1 flex-col justify-center gap-1">
        <div class="flex items-center gap-1.5 text-xs">
          <span
            :class="
              cn(
                'flex shrink-0 items-center gap-1 font-medium',
                ACCENT[card.kind]
              )
            "
          >
            <KunIcon :name="card.icon" class="size-3.5" />
            {{ card.noun }}
          </span>
          <span
            v-for="badge in card.badges"
            :key="badge.label"
            :class="
              cn(
                'rounded px-1 text-[10px] leading-4 font-medium',
                BADGE_TONE[badge.tone]
              )
            "
          >
            {{ badge.label }}
          </span>
        </div>

        <div class="flex min-w-0 items-baseline gap-2">
          <span
            :class="
              cn(
                'text-base font-semibold',
                card.kind === 'topic' ? 'line-clamp-2' : 'truncate',
                card.isUnavailable ? 'text-default-500' : 'text-foreground'
              )
            "
          >
            {{ card.title }}
          </span>
          <span
            v-if="card.layout === 'compact' && card.subtitle"
            class="text-default-500 truncate text-sm"
          >
            {{ card.subtitle }}
          </span>
          <span
            v-if="card.score !== null"
            class="text-warning-600 ml-auto shrink-0 text-xl leading-none font-bold"
          >
            {{ card.score }}<span class="text-default-400 text-xs">/10</span>
          </span>
        </div>

        <span
          v-if="card.layout === 'full' && card.subtitle"
          class="text-default-500 truncate text-sm"
        >
          {{ card.subtitle }}
        </span>
        <span
          v-if="card.description"
          :class="
            cn(
              'text-default-600 text-sm',
              card.layout === 'full' ? 'line-clamp-2' : 'line-clamp-1'
            )
          "
        >
          {{ card.description }}
        </span>

        <div
          v-if="card.chips.length"
          class="flex h-5 flex-wrap gap-1 overflow-hidden"
        >
          <span
            v-for="chip in card.chips"
            :key="chip"
            class="bg-default-100 text-default-600 rounded px-1.5 text-xs leading-5"
          >
            {{ chip }}
          </span>
        </div>

        <div
          v-if="card.person || card.info"
          class="text-default-500 flex min-w-0 items-center gap-1.5 text-xs"
        >
          <template v-if="card.person">
            <KunAvatar
              :user="card.person"
              :is-navigation="false"
              size="xs"
              class-name="shrink-0"
            />
            <span class="shrink-0">{{ card.person.name }}</span>
            <span v-if="card.info">·</span>
          </template>
          <span class="truncate">{{ card.info }}</span>
        </div>
      </div>
    </KunCard>
  </div>
</template>

<style scoped>
/* Prose and the editor both style every img and a in their content (1.75em
   image margins, link underline); inside a card those land on the cover and
   pushed it out of its frame in the editor on 2026-10-27. */
.kun-content-entity :deep(img) {
  margin: 0;
  max-width: none;
  border-radius: 0;
}

.kun-content-entity :deep(a) {
  color: inherit;
  font-weight: inherit;
  text-decoration: none;
}
</style>
