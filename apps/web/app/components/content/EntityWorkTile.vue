<script setup lang="ts">
import type { WorkRef } from '#shared/utils/api/schemas'
import { catalogNameText } from '~/utils/catalogName'

const props = defineProps<{ work: WorkRef }>()
const emit = defineEmits<{ navigate: [] }>()

const { showKUNGalgamePreferOriginalName } = storeToRefs(
  usePersistSettingsStore()
)
const { stance } = useContentStance()

const name = computed(() =>
  catalogNameText(props.work, showKUNGalgamePreferOriginalName.value)
)
const sensitive = computed(
  () =>
    props.work.is_nsfw ||
    props.work.cover?.sexual === 'suggestive' ||
    props.work.cover?.sexual === 'explicit'
)
const showCover = computed(
  () => props.work.cover && !(sensitive.value && stance.value === 'hide')
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
        v-if="showCover"
        :src="work.cover!.url"
        :alt="name"
        loading="lazy"
        :thumbhash="work.cover!.thumbhash ?? undefined"
        object-fit="cover"
        class-name="size-full"
        :image-class-name="
          cn(
            'size-full transition-transform group-hover:scale-105',
            sensitive && stance === 'blur' && 'blur-lg'
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
