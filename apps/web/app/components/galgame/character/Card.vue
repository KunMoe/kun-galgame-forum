<script setup lang="ts">
import type { CharacterSummary } from '#shared/utils/api/schemas'
import { traitSide } from '~/utils/galgame/trait'

const props = defineProps<{
  character: CharacterSummary
}>()

const nameOf = useCatalogName()
const { allowsNsfw, isBlurred } = useContentStance()

const name = computed(() => nameOf(props.character))
const image = computed(() => props.character.image)

const isExplicit = computed(() => image.value?.sexual === 'explicit')
const isHidden = computed(() => isExplicit.value && !allowsNsfw.value)
const isMasked = computed(() => isExplicit.value && isBlurred.value)
</script>

<template>
  <KunCard
    :href="`/galgame/character/${character.id}`"
    :is-hoverable="true"
    :is-transparent="false"
    padding="none"
    class-name="overflow-hidden"
  >
    <div class="bg-default-100 relative aspect-[3/4] w-full overflow-hidden">
      <KunImage
        v-if="image && !isHidden"
        :src="image.url"
        :alt="name.name"
        loading="lazy"
        :thumbhash="image.thumbhash ?? undefined"
        aspect-ratio="3 / 4"
        object-fit="cover"
        class-name="size-full"
        :image-class-name="cn('object-top', isMasked && 'scale-110 blur-xl')"
      />
      <div
        v-else
        class="text-default-400 flex size-full flex-col items-center justify-center gap-1"
      >
        <KunIcon
          :name="isHidden ? 'lucide:eye-off' : 'lucide:user-round'"
          class="size-8"
        />
        <span v-if="isHidden" class="text-xs">NSFW 头像已隐藏</span>
      </div>
    </div>

    <div class="space-y-0.5 p-2">
      <p class="truncate text-sm font-medium" :title="name.name">
        {{ name.name }}
      </p>
      <p
        v-if="name.original && name.original !== name.name"
        class="text-default-500 truncate text-xs"
        :title="name.original"
      >
        {{ name.original }}
      </p>
      <p
        v-if="character.catalog_work_count"
        class="text-default-400 text-xs tabular-nums"
      >
        登场 {{ character.catalog_work_count }} 部作品
      </p>
      <div
        v-if="character.matched_traits.length"
        class="flex flex-wrap gap-1 pt-1"
      >
        <KunChip
          v-for="trait in character.matched_traits"
          :key="trait.id"
          size="xs"
          variant="flat"
          color="primary"
        >
          {{ catalogVocabularyName(trait) }}
          <span v-if="traitSide(trait)" class="opacity-60">
            · {{ traitSide(trait) }}
          </span>
        </KunChip>
      </div>
    </div>
  </KunCard>
</template>
