<script setup lang="ts">
import type { WorkRef } from '#shared/utils/api/schemas'
import { catalogNameText } from '~/utils/catalogName'
import { explicitImage, maskCover } from '~/utils/entityCardModel'

const props = defineProps<{ work: WorkRef }>()
const emit = defineEmits<{ navigate: [] }>()

const { showKUNGalgamePreferOriginalName } = storeToRefs(
  usePersistSettingsStore()
)
const { stance } = useContentStance()
const blurAll = useEntityCoverBlur()

const name = computed(() =>
  catalogNameText(props.work, showKUNGalgamePreferOriginalName.value)
)
const sensitive = computed(
  () => props.work.is_nsfw || explicitImage(props.work.cover)
)
const showCover = computed(
  () => props.work.cover && !(sensitive.value && stance.value === 'hide')
)
const cover = computed(() =>
  props.work.cover
    ? maskCover(
        props.work.cover,
        props.work.cover_sfw,
        sensitive.value,
        stance.value,
        blurAll.value
      )
    : null
)
</script>

<template>
  <KunLink
    :to="`/galgame/${work.id}`"
    underline="none"
    color="default"
    class-name="group flex flex-col gap-1.5"
    @click="emit('navigate')"
  >
    <div
      class="bg-default-100 relative aspect-[5/7] w-full overflow-hidden rounded-lg"
    >
      <KunImage
        v-if="showCover && cover"
        :src="cover.image.url"
        :alt="name"
        loading="lazy"
        :thumbhash="cover.image.thumbhash ?? undefined"
        object-fit="cover"
        class-name="size-full"
        :image-class-name="
          cn(
            'size-full transition-transform group-hover:scale-105',
            cover.isBlurred && 'blur-lg'
          )
        "
      />
      <div
        v-else
        class="text-default-400 flex size-full items-center justify-center"
      >
        <KunIcon name="lucide:gamepad-2" class="size-8" />
      </div>
    </div>
    <span class="group-hover:text-primary line-clamp-2 text-xs leading-snug">
      {{ name }}
    </span>
  </KunLink>
</template>
