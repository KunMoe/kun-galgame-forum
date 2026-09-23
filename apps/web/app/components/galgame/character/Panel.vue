<script setup lang="ts">
import {
  KUN_GALGAME_CHARACTER_KIND_MAP,
  KUN_GALGAME_CHARACTER_KIND_COLOR,
  KUN_GALGAME_CHARACTER_SPOILER_MAP
} from '~/constants/galgameCharacter'
import type { Image, WorkCharacter } from '#shared/utils/api/schemas'
import type { GalgameArtMeta } from '~~/shared/types/galgame'

const props = defineProps<{
  roster: WorkCharacter[]
}>()

const nameOf = useCatalogName()

const spoilerRank = (spoiler: WorkCharacter['spoiler']) =>
  spoiler === 'major' ? 2 : spoiler === 'minor' ? 1 : 0

const isSpoilerRevealed = ref(false)
const visible = computed(() =>
  isSpoilerRevealed.value
    ? props.roster
    : props.roster.filter((c) => c.spoiler === 'none')
)
const spoilerCount = computed(
  () => props.roster.filter((c) => c.spoiler !== 'none').length
)

const featured = computed(() => visible.value.filter((c) => !!c.figure))
const portraits = computed(() =>
  visible.value.filter((c) => !c.figure && !!c.image)
)
const nameOnly = computed(() =>
  visible.value.filter((c) => !c.figure && !c.image)
)

const secondaryName = (c: WorkCharacter) => {
  const { name, original } = nameOf(c)
  return [original, c.latin].find((part) => !!part && part !== name) ?? ''
}

const thumbOf = (image: Image) => withImageVariant(image.url, 'mini')

const metaOf = (img: Image | null): GalgameArtMeta | undefined =>
  img?.width && img.height
    ? { width: img.width, height: img.height, thumbhash: img.thumbhash ?? '' }
    : undefined

const figureRatio = computed(() =>
  artGridRatio(
    featured.value.map((c) => metaOf(c.figure)),
    '1/1'
  )
)

const COLLAPSED_PORTRAITS = 12
const isExpanded = ref(false)
const isCollapsible = computed(
  () => portraits.value.length > COLLAPSED_PORTRAITS
)
const visiblePortraits = computed(() =>
  isCollapsible.value && !isExpanded.value
    ? portraits.value.slice(0, COLLAPSED_PORTRAITS)
    : portraits.value
)

const kindText = (kind: string) => KUN_GALGAME_CHARACTER_KIND_MAP[kind] || ''
const kindColor = (kind: string) =>
  KUN_GALGAME_CHARACTER_KIND_COLOR[kind] || 'default'

const activeCharacter = ref<WorkCharacter | null>(null)
const isModalOpen = ref(false)
const open = (character: WorkCharacter) => {
  activeCharacter.value = character
  isModalOpen.value = true
}

const voiceName = (voice: WorkCharacter['voices'][number]) =>
  nameOf(voice).name
</script>

<template>
  <div v-if="roster.length" class="space-y-4">
    <div class="flex flex-wrap items-end justify-between gap-2">
      <KunHeader
        name="登场角色"
        description="该 Galgame 的登场角色与配音演员, 数据来自 NextMoe 目录的角色图谱"
        scale="h2"
      />

      <KunButton
        v-if="spoilerCount"
        variant="flat"
        color="warning"
        size="sm"
        @click="isSpoilerRevealed = !isSpoilerRevealed"
      >
        <KunIcon :name="isSpoilerRevealed ? 'lucide:eye-off' : 'lucide:eye'" />
        {{
          isSpoilerRevealed
            ? `隐藏 ${spoilerCount} 名剧透角色`
            : `显示 ${spoilerCount} 名剧透角色`
        }}
      </KunButton>
    </div>

    <div
      v-if="featured.length"
      class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4"
    >
      <button
        v-for="c in featured"
        :key="c.id"
        type="button"
        class="group bg-default-100 hover:ring-primary focus:ring-primary flex flex-col overflow-hidden rounded-xl text-left ring-1 ring-transparent transition-all focus:outline-none"
        :aria-label="`查看角色 ${nameOf(c).name}`"
        @click="open(c)"
      >
        <div class="relative w-full">
          <KunImage
            :src="thumbOf(c.figure!)"
            :alt="nameOf(c).name"
            loading="lazy"
            :aspect-ratio="figureRatio"
            :thumbhash="c.figure?.thumbhash ?? undefined"
            object-fit="contain"
            class-name="w-full"
            image-class-name="transition-transform duration-200 group-hover:scale-105"
          />

          <KunChip
            v-if="kindText(c.character_kind)"
            :color="kindColor(c.character_kind)"
            size="xs"
            class-name="absolute top-2 left-2"
          >
            {{ kindText(c.character_kind) }}
          </KunChip>
        </div>

        <div class="bg-default-50 w-full space-y-0.5 px-3 py-2">
          <p class="text-default-800 truncate font-medium">
            {{ nameOf(c).name }}
          </p>
          <p v-if="secondaryName(c)" class="text-default-400 truncate text-xs">
            {{ secondaryName(c) }}
          </p>
          <p v-if="c.voices.length" class="text-default-500 truncate text-xs">
            CV {{ c.voices.map(voiceName).join(' / ') }}
          </p>
          <p
            v-if="KUN_GALGAME_CHARACTER_SPOILER_MAP[spoilerRank(c.spoiler)]"
            class="text-warning text-xs"
          >
            {{ KUN_GALGAME_CHARACTER_SPOILER_MAP[spoilerRank(c.spoiler)] }}
          </p>
        </div>
      </button>
    </div>

    <div
      v-if="visiblePortraits.length"
      class="grid grid-cols-3 gap-3 sm:grid-cols-[repeat(auto-fill,minmax(120px,1fr))]"
    >
      <button
        v-for="c in visiblePortraits"
        :key="c.id"
        type="button"
        class="group space-y-1.5 text-left"
        :aria-label="`查看角色 ${nameOf(c).name}`"
        @click="open(c)"
      >
        <div
          class="bg-default-100 group-hover:ring-primary group-focus:ring-primary relative overflow-hidden rounded-lg ring-1 ring-transparent transition-all"
        >
          <KunImage
            :src="thumbOf(c.image!)"
            :alt="nameOf(c).name"
            loading="lazy"
            aspect-ratio="3/4"
            :thumbhash="c.image?.thumbhash ?? undefined"
            object-fit="cover"
            class-name="w-full"
            image-class-name="transition-transform duration-200 group-hover:scale-105"
          />

          <KunChip
            v-if="kindText(c.character_kind)"
            :color="kindColor(c.character_kind)"
            size="xs"
            class-name="absolute top-1 left-1"
          >
            {{ kindText(c.character_kind) }}
          </KunChip>
        </div>

        <div class="space-y-0.5">
          <p class="text-default-800 truncate text-sm font-medium">
            {{ nameOf(c).name }}
          </p>
          <p
            v-if="c.voices.length"
            class="text-default-500 truncate text-xs"
            :title="c.voices.map(voiceName).join(' / ')"
          >
            CV {{ c.voices.map(voiceName).join(' / ') }}
          </p>
          <p
            v-if="KUN_GALGAME_CHARACTER_SPOILER_MAP[spoilerRank(c.spoiler)]"
            class="text-warning text-xs"
          >
            {{ KUN_GALGAME_CHARACTER_SPOILER_MAP[spoilerRank(c.spoiler)] }}
          </p>
        </div>
      </button>
    </div>

    <KunButton
      v-if="isCollapsible"
      variant="flat"
      color="primary"
      size="sm"
      @click="isExpanded = !isExpanded"
    >
      <KunIcon
        :name="isExpanded ? 'lucide:chevron-up' : 'lucide:chevron-down'"
      />
      {{
        isExpanded
          ? '收起角色'
          : `展开其余 ${portraits.length - COLLAPSED_PORTRAITS} 名角色`
      }}
    </KunButton>

    <div v-if="nameOnly.length" class="space-y-1.5">
      <p
        v-if="featured.length || visiblePortraits.length"
        class="text-default-500 text-sm"
      >
        其他登场角色
      </p>
      <div class="flex flex-wrap items-baseline gap-x-4 gap-y-1.5">
        <span v-for="c in nameOnly" :key="c.id" class="text-sm">
          <button
            type="button"
            class="text-default-800 hover:text-primary cursor-pointer"
            @click="open(c)"
          >
            {{ nameOf(c).name }}
          </button>
          <span v-if="c.voices.length" class="text-default-400">
            （CV
            <template v-for="(v, index) in c.voices" :key="v.id">
              <span v-if="index"> / </span>
              <KunLink
                :to="`/galgame/staff/${v.id}`"
                underline="none"
                size="sm"
                class-name="text-default-400 hover:text-primary"
              >
                {{ voiceName(v) }}
              </KunLink>
            </template>
            ）
          </span>
        </span>
      </div>
    </div>

    <GalgameCharacterModal v-model="isModalOpen" :character="activeCharacter" />
  </div>
</template>
