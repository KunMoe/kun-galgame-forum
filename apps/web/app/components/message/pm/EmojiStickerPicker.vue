<script setup lang="ts">
import { emojiArray } from '~/constants/emoji'

const emit = defineEmits<{
  emoji: [emoji: string]
  sticker: [url: string]
}>()

const tab = ref<'emoji' | 'sticker'>('emoji')

// Loaded on first open of the sticker tab, not on mount: most messages are
// sent without one, and the payload is 82 KB.
const { packs, load } = useStickerPacks()
const stickers = computed(() => packs.value.flatMap((pack) => pack.stickers))
watch(tab, (value) => {
  if (value === 'sticker') load()
})
</script>

<template>
  <div class="w-72 p-2 sm:w-80">
    <div class="bg-default-100 mb-2 flex rounded-full p-1 text-sm">
      <button
        type="button"
        @click="tab = 'emoji'"
        :class="
          cn(
            'flex-1 rounded-full py-1.5 transition-colors',
            tab === 'emoji'
              ? 'bg-background text-primary font-medium shadow-sm'
              : 'text-default-500 hover:text-default-700'
          )
        "
      >
        表情
      </button>
      <button
        type="button"
        @click="tab = 'sticker'"
        :class="
          cn(
            'flex-1 rounded-full py-1.5 transition-colors',
            tab === 'sticker'
              ? 'bg-background text-primary font-medium shadow-sm'
              : 'text-default-500 hover:text-default-700'
          )
        "
      >
        贴纸
      </button>
    </div>

    <KunOverlayScroll v-show="tab === 'emoji'" class="h-56">
      <div class="grid grid-cols-8 gap-0.5">
        <button
          v-for="(e, i) in emojiArray"
          :key="i"
          type="button"
          @click="emit('emoji', e)"
          class="hover:bg-default-100 flex aspect-square items-center justify-center rounded-md text-xl"
        >
          {{ e }}
        </button>
      </div>
    </KunOverlayScroll>

    <KunOverlayScroll v-show="tab === 'sticker'" class="h-56">
      <div class="grid grid-cols-4 gap-1">
        <button
          v-for="sticker in stickers"
          :key="sticker.src"
          type="button"
          @click="emit('sticker', sticker.src)"
          class="hover:bg-default-100 aspect-square rounded-md p-1"
        >
          <img
            :src="sticker.src"
            :alt="sticker.name"
            loading="lazy"
            class="size-full object-contain"
          />
        </button>
      </div>
    </KunOverlayScroll>
  </div>
</template>
