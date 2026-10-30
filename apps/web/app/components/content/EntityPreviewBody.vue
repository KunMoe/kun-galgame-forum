<script setup lang="ts">
import type { EntityTarget } from '#shared/utils/api/schemas'
import { resourceTypeLabel } from '#shared/utils/galgameResourceVocab'
import { entityCardModel } from '~/utils/entityCardModel'
import { aspectDims } from '~/utils/galgame/ratingCard'

const props = defineProps<{
  target: EntityTarget
}>()

const emit = defineEmits<{ navigate: [] }>()

const { showKUNGalgamePreferOriginalName } = storeToRefs(
  usePersistSettingsStore()
)
const { stance } = useContentStance()
const { detail, status } = useEntityPreviewDetail(() => props.target)

const card = computed(() =>
  entityCardModel(
    props.target,
    showKUNGalgamePreferOriginalName.value,
    stance.value
  )
)

const isIntroOpen = ref(false)
const isSpoilerShown = ref(false)
watch(
  () => props.target,
  () => {
    isIntroOpen.value = false
    isSpoilerShown.value = false
  }
)

const dims = computed(() =>
  detail.value.rating ? aspectDims(detail.value.rating.aspect_scores) : null
)
const hasDims = computed(
  () => !!dims.value && Object.values(dims.value).some((v) => v > 0)
)

const isRatingMasked = computed(
  () =>
    (detail.value.rating?.spoiler_level ?? 'none') !== 'none' &&
    !isSpoilerShown.value
)
</script>

<template>
  <div class="flex flex-col gap-6">
    <header class="flex gap-4">
      <ContentEntityCardMedia
        :media="card.media"
        accent="text-default-500"
        :top-anchored="card.kind === 'character'"
      />
      <div class="flex min-w-0 flex-1 flex-col justify-center gap-1.5">
        <span class="text-default-500 flex items-center gap-1 text-xs">
          <KunIcon :name="card.icon" class="size-3.5" />
          {{ card.noun }}
        </span>
        <h3 class="text-lg leading-snug font-bold break-words">
          {{ card.title }}
          <span
            v-if="card.score !== null"
            class="text-warning-600 ml-1 text-xl"
          >
            {{ card.score }}<span class="text-default-400 text-xs">/10</span>
          </span>
        </h3>
        <p v-if="card.subtitle" class="text-default-500 text-sm">
          {{ card.subtitle }}
        </p>
        <div v-if="card.chips.length" class="flex flex-wrap gap-1">
          <span
            v-for="chip in card.chips"
            :key="chip"
            class="bg-default-100 text-default-600 rounded px-1.5 text-xs leading-5"
          >
            {{ chip }}
          </span>
        </div>
        <p
          v-if="card.person || card.info"
          class="text-default-500 flex flex-wrap items-center gap-1.5 text-xs"
        >
          <template v-if="card.person">
            <KunAvatar :user="card.person" size="xs" />
            {{ card.person.name }}
            <span v-if="card.info">·</span>
          </template>
          {{ card.info }}
        </p>
      </div>
    </header>

    <div v-if="status === 'loading'" class="space-y-2">
      <div class="bg-default-100 h-4 w-full animate-pulse rounded" />
      <div class="bg-default-100 h-4 w-5/6 animate-pulse rounded" />
      <div class="bg-default-100 h-4 w-2/3 animate-pulse rounded" />
    </div>

    <template v-else>
      <section v-if="detail.intro" class="space-y-2">
        <p
          :class="
            cn(
              'text-default-700 text-sm leading-relaxed whitespace-pre-line',
              !isIntroOpen && 'line-clamp-6'
            )
          "
        >
          {{ detail.intro }}
        </p>
        <KunButton
          v-if="detail.intro.length > 160"
          size="sm"
          variant="light"
          @click="isIntroOpen = !isIntroOpen"
        >
          {{ isIntroOpen ? '收起' : '展开全部' }}
        </KunButton>
      </section>

      <section v-if="detail.rating?.short_summary" class="space-y-2">
        <ContentSpoilerText
          v-if="!isRatingMasked"
          :content="detail.rating.short_summary"
        />
        <KunButton
          v-else
          size="sm"
          variant="flat"
          color="warning"
          @click="isSpoilerShown = true"
        >
          <KunIcon name="lucide:eye" />
          这篇评分含有剧透，点击查看
        </KunButton>
      </section>

      <section v-if="detail.resource">
        <GalgameResourceDetailInfo
          :resource="detail.resource"
          :resource-type-label="
            resourceTypeLabel(detail.resource.resource_type)
          "
          :refresh="() => {}"
        />
      </section>

      <section v-if="hasDims" class="flex justify-center">
        <GalgameRatingRadar :model-value="dims!" readonly :size="260" />
      </section>

      <section
        v-if="detail.document && detail.document.children.length"
        class="border-default-200 max-h-[50dvh] overflow-y-auto rounded-lg border p-4"
      >
        <ContentDocument :document="detail.document" compact />
      </section>

      <section
        v-if="
          detail.works.length || detail.topics.length || detail.resources.length
        "
        class="space-y-3"
      >
        <h4 class="text-sm font-semibold">{{ detail.relatedTitle }}</h4>

        <div v-if="detail.works.length" class="grid grid-cols-3 gap-3">
          <ContentEntityWorkTile
            v-for="work in detail.works"
            :key="work.id"
            :work="work"
            @navigate="emit('navigate')"
          />
        </div>

        <ul v-if="detail.topics.length" class="space-y-1">
          <li v-for="topic in detail.topics" :key="topic.id">
            <KunLink
              :to="`/topic/${topic.id}`"
              underline="none"
              color="default"
              class-name="hover:bg-default-100 flex items-center justify-between gap-3 rounded-lg px-2 py-2 text-sm"
              @click="emit('navigate')"
            >
              <span class="line-clamp-1">{{ topic.title }}</span>
              <KunTime
                :time="topic.created_at"
                type="relative"
                class="text-default-500 shrink-0 text-xs"
              />
            </KunLink>
          </li>
        </ul>

        <ul v-if="detail.resources.length" class="space-y-1">
          <li v-for="res in detail.resources" :key="res.id">
            <KunLink
              :to="`/galgame/resource/${res.id}`"
              underline="none"
              color="default"
              class-name="hover:bg-default-100 flex items-center justify-between gap-3 rounded-lg px-2 py-2 text-sm"
              @click="emit('navigate')"
            >
              <span class="line-clamp-1">
                {{ res.title || resourceTypeLabel(res.resource_type) }}
              </span>
              <span class="text-default-500 shrink-0 text-xs">
                {{ resourceTypeLabel(res.resource_type) }} · {{ res.size }}
              </span>
            </KunLink>
          </li>
        </ul>
      </section>
    </template>
  </div>
</template>
