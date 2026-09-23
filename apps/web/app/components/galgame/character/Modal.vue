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

const props = defineProps<{
  character: GalgameDetailCharacter | null
}>()

const isOpen = defineModel<boolean>({ required: true })

const api = useApiClient()
const nameOf = useCatalogName()
const { allowsNsfw } = useContentStance()

const cache = new Map<number, CharacterView>()
const detail = ref<CharacterView | null>(null)
const isLoading = ref(false)

const load = async (id: number) => {
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
        path: { character_id: String(id) },
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

const figureFrame = computed(() =>
  artFrame(props.character?.figure_meta, detail.value?.figure_meta)
)
const bustFrame = computed(() =>
  artFrame(props.character?.image_meta, detail.value?.image_meta)
)

const kindText = computed(() =>
  props.character
    ? KUN_GALGAME_CHARACTER_KIND_MAP[props.character.kind] || ''
    : ''
)
const kindColor = computed(() =>
  props.character
    ? KUN_GALGAME_CHARACTER_KIND_COLOR[props.character.kind] || 'default'
    : 'default'
)
const spoilerText = computed(() =>
  props.character
    ? KUN_GALGAME_CHARACTER_SPOILER_MAP[props.character.spoiler]
    : ''
)

const isTraitSpoilerRevealed = ref(false)
const traits = computed(() => {
  const all = detail.value?.traits ?? []
  return isTraitSpoilerRevealed.value ? all : all.filter((t) => t.spoiler === 0)
})
const hiddenTraitCount = computed(
  () => (detail.value?.traits ?? []).filter((t) => t.spoiler > 0).length
)

const traitGroups = computed(() => {
  const groups: { name: string; traits: GalgameCharacterTrait[] }[] = []
  for (const trait of traits.value) {
    const name = trait.group || '其他'
    const last = groups.at(-1)
    if (last && last.name === name) {
      last.traits.push(trait)
    } else {
      groups.push({ name, traits: [trait] })
    }
  }
  return groups
})

const introCredit = computed(() =>
  getGalgameCharacterIntroCredit(
    detail.value?.intros.find((i) => i.intro === detail.value?.intro)
  )
)

const heading = computed(
  () => detail.value?.name || props.character?.name || ''
)
const headingOriginal = computed(() => {
  const parts = [
    detail.value?.name_original ?? props.character?.name_original,
    props.character?.latin
  ].filter((part): part is string => !!part && part !== heading.value)
  return parts.join(' · ')
})

watch(
  () => props.character?.id,
  () => {
    isTraitSpoilerRevealed.value = false
  }
)
</script>

<template>
  <KunModal v-model="isOpen" size="xl" scroll-behavior="inside">
    <div v-if="character" class="space-y-4">
      <div class="flex flex-col gap-4 sm:flex-row">
        <KunLightboxGallery v-if="character.figure || character.image">
          <div
            class="flex shrink-0 items-start gap-3 sm:flex-col sm:items-center"
          >
            <KunLightboxGalleryItem
              v-if="character.figure"
              :src="character.figure"
              :alt="character.name"
              :wrap="false"
              v-slot="{ open }"
            >
              <button
                type="button"
                class="bg-default-100 w-fit cursor-zoom-in overflow-hidden rounded-xl"
                :aria-label="`查看 ${character.name} 的立绘`"
                @click="open"
              >
                <KunImage
                  :src="character.figure"
                  :alt="character.name"
                  loading="eager"
                  :aspect-ratio="figureFrame.aspectRatio"
                  :object-fit="figureFrame.objectFit"
                  :thumbhash="figureFrame.thumbhash"
                  class-name="w-36 sm:w-48"
                />
              </button>
            </KunLightboxGalleryItem>

            <KunLightboxGalleryItem
              v-if="character.image"
              :src="character.image"
              :alt="character.name"
              :wrap="false"
              v-slot="{ open }"
            >
              <button
                type="button"
                class="bg-default-100 w-fit cursor-zoom-in overflow-hidden rounded-xl"
                :aria-label="`查看 ${character.name} 的头像`"
                @click="open"
              >
                <KunImage
                  :src="character.image"
                  :alt="character.name"
                  loading="eager"
                  :aspect-ratio="bustFrame.aspectRatio"
                  :object-fit="bustFrame.objectFit"
                  :thumbhash="bustFrame.thumbhash"
                  :class-name="
                    character.figure ? 'w-20 sm:w-24' : 'w-36 sm:w-48'
                  "
                />
              </button>
            </KunLightboxGalleryItem>
          </div>
        </KunLightboxGallery>

        <div class="min-w-0 grow space-y-3">
          <div class="space-y-1">
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
                {{ v.name }}
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

            <div v-if="traitGroups.length" class="space-y-2">
              <div
                v-for="group in traitGroups"
                :key="group.name"
                class="space-y-1"
              >
                <p class="text-default-400 text-xs">{{ group.name }}</p>
                <div class="flex flex-wrap gap-1.5">
                  <KunChip
                    v-for="trait in group.traits"
                    :key="trait.id"
                    size="xs"
                    :color="trait.spoiler > 0 ? 'warning' : 'default'"
                  >
                    {{ trait.name }}<template v-if="trait.lie">（伪）</template>
                  </KunChip>
                </div>
              </div>
            </div>

            <KunButton
              v-if="hiddenTraitCount && !isTraitSpoilerRevealed"
              variant="flat"
              color="warning"
              size="sm"
              @click="isTraitSpoilerRevealed = true"
            >
              <KunIcon name="lucide:eye" />
              显示 {{ hiddenTraitCount }} 条剧透特征
            </KunButton>

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
  </KunModal>
</template>
