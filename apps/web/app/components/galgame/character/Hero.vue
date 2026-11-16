<script setup lang="ts">
import {
  getGalgameCharacterIntroCredit,
  getGalgameCharacterLangName,
  getGalgameCharacterVoiceLabel
} from '~/constants/galgameCharacter'
import type { CharacterView } from '~/utils/galgame/entityCards'

const props = defineProps<{
  character: CharacterView
  voices: GalgameDetailCharacterVoice[]
  workCountText: string
}>()

const emit = defineEmits<{
  openArt: [kind: 'image' | 'figure']
}>()

const VOICE_PREVIEW = 3

const subtitle = computed(() =>
  [props.character.name_original, props.character.latin]
    .filter((part) => !!part && part !== props.character.name)
    .join(' · ')
)

const heroArt = computed(() =>
  props.character.image
    ? {
        kind: 'image' as const,
        src: props.character.image,
        meta: props.character.image_meta
      }
    : props.character.figure
      ? {
          kind: 'figure' as const,
          src: props.character.figure,
          meta: props.character.figure_meta
        }
      : null
)

const intros = computed(() => {
  const all = props.character.intros
  const lead = all.find((i) => i.intro === props.character.intro)
  return lead ? [lead, ...all.filter((i) => i !== lead)] : all
})
const introTabs = computed(() =>
  intros.value.map((i) => ({
    value: i.lang,
    textValue: getGalgameCharacterLangName(i.lang)
  }))
)
const introLang = ref(intros.value[0]?.lang ?? '')
watch(intros, (rows) => {
  if (!rows.some((i) => i.lang === introLang.value)) {
    introLang.value = rows[0]?.lang ?? ''
  }
})
</script>

<template>
  <GalgameEntityHero :name="character.name" :subtitle="subtitle">
    <template v-if="heroArt" #media>
      <div class="sm:hidden">
        <GalgameCharacterArt
          :src="heroArt.src"
          :alt="character.name"
          :label="`查看 ${character.name} 的图片`"
          :meta="heroArt.meta"
          :max-width="80"
          :max-height="112"
          @open="emit('openArt', heroArt.kind)"
        />
      </div>
      <div class="hidden sm:block">
        <GalgameCharacterArt
          :src="heroArt.src"
          :alt="character.name"
          :label="`查看 ${character.name} 的图片`"
          :meta="heroArt.meta"
          :max-width="144"
          :max-height="200"
          @open="emit('openArt', heroArt.kind)"
        />
      </div>
    </template>

    <template #meta>
      <p>
        登场
        <span class="text-foreground font-semibold tabular-nums">
          {{ workCountText }}
        </span>
        部作品
      </p>
      <p v-if="voices.length" class="mt-1">
        CV
        <template
          v-for="(voice, index) in voices.slice(0, VOICE_PREVIEW)"
          :key="voice.id"
        >
          <template v-if="index"> / </template>
          <KunLink
            :to="`/galgame/staff/${voice.id}`"
            size="sm"
            underline="hover"
            :title="voice.latin || voice.name"
          >
            {{ getGalgameCharacterVoiceLabel(voice) }}
          </KunLink>
        </template>
        <template v-if="voices.length > VOICE_PREVIEW">
          等 {{ voices.length }} 人
        </template>
      </p>
    </template>

    <template
      v-if="(character.image && character.figure) || character.links.length"
      #actions
    >
      <div class="flex flex-wrap items-center gap-x-4 gap-y-2">
        <KunButton
          v-if="character.image && character.figure"
          class-name="lg:hidden"
          variant="flat"
          color="primary"
          size="sm"
          @click="emit('openArt', 'figure')"
        >
          <KunIcon name="lucide:user-round" />
          查看立绘
        </KunButton>
        <GalgameEntityLinks :links="character.links" />
      </div>
    </template>

    <div v-if="intros.length" class="space-y-2">
      <KunTab
        v-if="introTabs.length > 1"
        v-model="introLang"
        :items="introTabs"
        name="character-intro"
        variant="underlined"
        size="sm"
      />
      <KunTabPanels v-model="introLang" name="character-intro">
        <KunTabPanel
          v-for="intro in intros"
          :key="intro.lang"
          :value="intro.lang"
        >
          <GalgameEntityIntro
            :text="intro.intro"
            :note="getGalgameCharacterIntroCredit(intro)"
          />
        </KunTabPanel>
      </KunTabPanels>
    </div>
  </GalgameEntityHero>
</template>
