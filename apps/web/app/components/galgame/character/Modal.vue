<script setup lang="ts">
import {
  KUN_GALGAME_CHARACTER_KIND_MAP,
  KUN_GALGAME_CHARACTER_KIND_COLOR,
  KUN_GALGAME_CHARACTER_SPOILER_MAP,
  getGalgameCharacterIntroCredit
} from '~/constants/galgameCharacter'
import { settle } from '#shared/utils/api/problem'
import {
  characterViewOf,
  type CharacterView
} from '~/utils/galgame/entityCards'
import type { Image, WorkCharacter } from '#shared/utils/api/schemas'
import type { GalgameArtMeta } from '~~/shared/types/galgame'

const props = defineProps<{
  character: WorkCharacter | null
}>()

const isOpen = defineModel<boolean>({ required: true })

const api = useApiClient()
const nameOf = useCatalogName()
const { allowsNsfw } = useContentStance()

const cache = new Map<string, CharacterView>()
const detail = ref<CharacterView | null>(null)
const isLoading = ref(false)

const load = async (id: string) => {
  const cached = cache.get(id)
  if (cached) {
    detail.value = cached
    return
  }
  detail.value = null
  isLoading.value = true
  const res = await settle(
    api.GET('/characters/{character_id}', {
      params: {
        path: { character_id: id },
        query: { include_nsfw: allowsNsfw.value }
      }
    })
  )
  isLoading.value = false
  if (!res.ok) {
    return
  }
  const view = characterViewOf(res.data, nameOf)
  cache.set(id, view)
  if (props.character?.id === id) {
    detail.value = view
  }
}

watch(
  () => [isOpen.value, props.character?.id] as const,
  ([open, id]) => {
    if (open && id) {
      load(id)
    }
  },
  { immediate: true }
)

const metaOf = (img: Image | null | undefined): GalgameArtMeta | undefined =>
  img?.width && img.height
    ? { width: img.width, height: img.height, thumbhash: img.thumbhash ?? '' }
    : undefined

const figureMeta = computed(
  () => metaOf(props.character?.figure) ?? detail.value?.figure_meta
)
const bustMeta = computed(
  () => metaOf(props.character?.image) ?? detail.value?.image_meta
)

const artImages = computed(() =>
  [props.character?.figure, props.character?.image]
    .filter((img): img is Image => !!img)
    .map((img) => ({ src: img.url, alt: heading.value }))
)
const bustIndex = computed(() => (props.character?.figure ? 1 : 0))
const isLightboxOpen = ref(false)
const lightboxIndex = ref(0)
const openArt = (index: number) => {
  lightboxIndex.value = index
  isLightboxOpen.value = true
}

const spoilerRank = (spoiler: WorkCharacter['spoiler'] | undefined) =>
  spoiler === 'major' ? 2 : spoiler === 'minor' ? 1 : 0

const kindText = computed(() =>
  props.character
    ? KUN_GALGAME_CHARACTER_KIND_MAP[props.character.character_kind] || ''
    : ''
)
const kindColor = computed(() =>
  props.character
    ? KUN_GALGAME_CHARACTER_KIND_COLOR[props.character.character_kind] ||
      'default'
    : 'default'
)
const spoilerText = computed(() =>
  props.character
    ? KUN_GALGAME_CHARACTER_SPOILER_MAP[spoilerRank(props.character.spoiler)]
    : ''
)

const introCredit = computed(() =>
  getGalgameCharacterIntroCredit(
    detail.value?.intros.find((i) => i.intro === detail.value?.intro)
  )
)

const heading = computed(
  () =>
    detail.value?.name ||
    (props.character ? nameOf(props.character).name : '') ||
    ''
)
const headingOriginal = computed(() => {
  const parts = [
    detail.value?.name_original ??
      (props.character ? nameOf(props.character).original : ''),
    props.character?.latin
  ].filter((part): part is string => !!part && part !== heading.value)
  return parts.join(' · ')
})
</script>

<template>
  <KunModal
    v-model="isOpen"
    size="xl"
    scroll-behavior="inside"
    :aria-label="heading"
  >
    <div v-if="character" class="space-y-4">
      <div class="flex gap-5">
        <div
          v-if="character.figure || character.image"
          class="hidden shrink-0 flex-col items-center gap-3 sm:flex"
        >
          <GalgameCharacterArt
            v-if="character.figure"
            :src="character.figure.url"
            :alt="heading"
            :label="`查看 ${heading} 的立绘`"
            :meta="figureMeta"
            :max-width="200"
            :max-height="360"
            @open="openArt(0)"
          />
          <GalgameCharacterArt
            v-if="character.image"
            :src="character.image.url"
            :alt="heading"
            :label="`查看 ${heading} 的头像`"
            :meta="bustMeta"
            :max-width="character.figure ? 96 : 180"
            :max-height="character.figure ? 112 : 240"
            @open="openArt(bustIndex)"
          />
        </div>

        <div class="min-w-0 grow space-y-3">
          <div class="flex items-start gap-3">
            <GalgameCharacterArt
              v-if="character.image"
              class="sm:hidden"
              :src="character.image.url"
              :alt="heading"
              :label="`查看 ${heading} 的头像`"
              :meta="bustMeta"
              :max-width="72"
              :max-height="96"
              @open="openArt(bustIndex)"
            />

            <div class="min-w-0 space-y-1">
              <div class="flex flex-wrap items-center gap-2">
                <h3 class="text-foreground text-xl font-medium">
                  {{ heading }}
                </h3>
                <KunChip v-if="kindText" :color="kindColor" size="sm">
                  {{ kindText }}
                </KunChip>
                <KunChip v-if="spoilerText" color="warning" size="sm">
                  {{ spoilerText }}
                </KunChip>
              </div>
              <p v-if="headingOriginal" class="text-default-400 text-sm">
                {{ headingOriginal }}
              </p>
              <KunButton
                v-if="character.figure"
                class-name="sm:hidden"
                variant="flat"
                color="primary"
                size="xs"
                @click="openArt(0)"
              >
                <KunIcon name="lucide:user-round" />
                查看立绘
              </KunButton>
            </div>
          </div>

          <div v-if="character.voices.length" class="text-default-500 text-sm">
            CV
            <template v-for="(v, index) in character.voices" :key="v.id">
              <span v-if="index"> / </span>
              <KunLink
                :to="`/galgame/staff/${v.id}`"
                underline="none"
                size="sm"
                class-name="text-default-600 hover:text-primary"
              >
                {{ nameOf(v).name }}
              </KunLink>
            </template>
          </div>

          <KunLoading v-if="isLoading" />

          <template v-else-if="detail">
            <div v-if="detail.intro" class="space-y-1">
              <p
                class="text-default-600 max-h-48 overflow-y-auto text-sm whitespace-pre-line"
              >
                {{ detail.intro }}
              </p>
              <p v-if="introCredit" class="text-default-400 text-xs">
                {{ introCredit }}
              </p>
            </div>

            <GalgameCharacterTraits :traits="detail.traits" />

            <div
              v-if="detail.links.length"
              class="flex flex-wrap items-center gap-3"
            >
              <template v-for="link in detail.links" :key="link.source">
                <KunLink
                  v-if="link.url"
                  :to="link.url"
                  target="_blank"
                  rel="noopener noreferrer"
                  size="sm"
                  color="default"
                  class-name="text-default-500 hover:text-default-700"
                >
                  {{ link.name }}
                  <KunIcon name="lucide:external-link" class="inline size-3" />
                </KunLink>
                <span v-else class="text-default-400 text-sm">
                  {{ link.name }}
                </span>
              </template>
            </div>
          </template>
        </div>
      </div>

      <div class="flex justify-end">
        <KunButton
          :href="`/galgame/character/${character.id}`"
          variant="flat"
          color="primary"
        >
          查看角色详情
          <KunIcon name="lucide:arrow-right" />
        </KunButton>
      </div>
    </div>

    <KunLightbox
      v-model:is-open="isLightboxOpen"
      :images="artImages"
      :initial-index="lightboxIndex"
    />
  </KunModal>
</template>
