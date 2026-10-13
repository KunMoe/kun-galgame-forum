<script setup lang="ts">
import type { RelatedWork, WorkMedium } from '#shared/utils/api/schemas'
import { relatedWorkToCard } from '~/utils/galgame/workCard'

const props = defineProps<{
  works: RelatedWork[]
}>()

const CARD_LIMIT = 12
const OTHER_LIMIT = 6

const MEDIUM_LABELS: Partial<Record<WorkMedium, string>> = {
  anime: '动画',
  manga: '漫画',
  novel: '小说',
  asmr: '音声',
  doujin_game: '同人游戏',
  music: '音乐'
}

const nameOf = useCatalogName()

const phrasesOf = computed(
  () =>
    new Map(
      props.works.map((w) => [
        Number(w.id),
        w.relations.map((r) => r.phrase).join(' · ')
      ])
    )
)

const games = computed(() =>
  props.works
    .filter((w) => w.medium === 'galgame')
    .map((w) => relatedWorkToCard(w, nameOf))
)
const others = computed(() =>
  props.works
    .filter((w) => w.medium !== 'galgame')
    .map((w) => `${nameOf(w).name}（${MEDIUM_LABELS[w.medium] ?? w.medium}）`)
)

const showAllGames = ref(false)
const showAllOthers = ref(false)
const shownGames = computed(() =>
  showAllGames.value ? games.value : games.value.slice(0, CARD_LIMIT)
)
const shownOthers = computed(() =>
  showAllOthers.value ? others.value : others.value.slice(0, OTHER_LIMIT)
)
</script>

<template>
  <div v-if="works.length" class="space-y-3">
    <KunHeader name="关联作品" scale="h3" />

    <GalgameCard v-if="shownGames.length" :galgames="shownGames">
      <template #meta="{ galgame }">
        <p class="text-default-500 mt-1 line-clamp-2 text-xs">
          {{ phrasesOf.get(galgame.id) }}
        </p>
      </template>
    </GalgameCard>
    <KunButton
      v-if="games.length > CARD_LIMIT && !showAllGames"
      variant="flat"
      size="sm"
      @click="showAllGames = true"
    >
      展开全部（{{ games.length }}）
    </KunButton>

    <div v-if="others.length" class="text-default-600 text-sm">
      <span class="text-default-500">其他媒介：</span>
      <span>{{ shownOthers.join(' · ') }}</span>
      <KunButton
        v-if="others.length > OTHER_LIMIT && !showAllOthers"
        variant="light"
        size="sm"
        @click="showAllOthers = true"
      >
        展开全部（{{ others.length }}）
      </KunButton>
    </div>
  </div>
</template>
