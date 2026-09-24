<script setup lang="ts">
import {
  KUN_GALGAME_PLAYTIME_SOURCE_CONST,
  KUN_GALGAME_PLAYTIME_SOURCE_MAP,
  KUN_GALGAME_PLAY_STATE_MAP,
  type KunGalgamePlayState,
  type KunGalgamePlayStateRead
} from '~/constants/galgame-playtime'
import type { Work, WorkViewerPlaytime } from '#shared/utils/api/schemas'

const props = defineProps<{
  galgame: Work
  hasLocalRating: boolean
}>()

const emits = defineEmits<{
  wantsRating: [KunGalgamePlayState]
}>()

const { id } = usePersistUserStore()

const { playtimeOf, ensureLoaded } = useMyGalgameInteractions()
onMounted(() => ensureLoaded([props.galgame.id]))
const saved = ref<WorkViewerPlaytime | null | undefined>(undefined)
// undefined is unknown — still loading, or catalog could not be read — and is
// never drawn as "no record".
const mine = computed(() =>
  saved.value !== undefined ? saved.value : playtimeOf(props.galgame.id)
)
const unknown = computed(() => !!id && mine.value === undefined)

const isOpen = ref(false)

const chips = computed(() =>
  KUN_GALGAME_PLAYTIME_SOURCE_CONST.flatMap((source) => {
    const row = props.galgame.playtimes.find((p) => p.site === source)
    if (!row) return []

    const meta = KUN_GALGAME_PLAYTIME_SOURCE_MAP[source]
    if (meta.hasVoteCount && row.vote_count < meta.minVotes) return []

    const duration = formatDurationMinutes(row.minutes)
    if (!duration) return []

    const votes = meta.hasVoteCount
      ? `${row.vote_count.toLocaleString('en-US')} 人`
      : ''
    return [
      {
        key: source,
        short: meta.short,
        duration,
        votes,
        tooltip: votes ? `${meta.hint}, 共 ${votes}` : meta.hint
      }
    ]
  })
)

const myDuration = computed(() =>
  mine.value && mine.value.minutes > 0
    ? formatDurationMinutes(mine.value.minutes)
    : ''
)

const myStatusLabel = computed(() => {
  if (!mine.value?.play_state) return ''
  return (
    KUN_GALGAME_PLAY_STATE_MAP[
      mine.value.play_state as KunGalgamePlayStateRead
    ] ?? mine.value.play_state
  )
})

const myTooltip = computed(() => {
  if (unknown.value) return '暂时读不到你的游玩记录'
  if (!mine.value) return '标记你在这部作品上的游玩状态, 也可以记下用时'
  const parts: string[] = []
  if (myStatusLabel.value && myDuration.value) {
    parts.push(`你的记录: ${myDuration.value} · ${myStatusLabel.value}`)
  } else if (myStatusLabel.value) {
    parts.push(`你的记录: ${myStatusLabel.value}`)
  } else if (myDuration.value) {
    parts.push(`你的记录: ${myDuration.value}`)
  }
  const site = props.galgame.playtimes.find((p) => p.site === 'nextmoe')
  if (site) {
    parts.push(`本站中位数 ${formatDurationMinutes(site.minutes)}`)
  }
  return parts.join(', ')
})

const openEditor = () => {
  if (!id) {
    useAuthModal().open()
    return
  }
  isOpen.value = true
}

const onFinished = (state: KunGalgamePlayState) => {
  if (!id) return
  if (props.hasLocalRating) return
  emits('wantsRating', state)
}
</script>

<template>
  <div v-if="chips.length || id" class="flex flex-wrap items-center gap-2">
    <span class="text-default-600 flex shrink-0 items-center gap-1.5 text-sm">
      <KunIcon name="lucide:clock" class="text-default-400" />
      游玩时长
    </span>

    <KunTooltip
      v-for="chip in chips"
      :key="chip.key"
      :text="chip.tooltip"
      class-name="shrink-0"
    >
      <KunChip size="sm" variant="flat" color="default">
        <span class="text-default-500 font-medium">{{ chip.short }}</span>
        <span class="tabular-nums">{{ chip.duration }}</span>
        <span v-if="chip.votes" class="text-default-400 tabular-nums">
          · {{ chip.votes }}
        </span>
      </KunChip>
    </KunTooltip>

    <KunTooltip :text="myTooltip" class-name="shrink-0">
      <KunButton
        size="sm"
        :variant="mine ? 'flat' : 'light'"
        color="primary"
        :disabled="unknown"
        @click="openEditor"
      >
        <KunIcon :name="mine ? 'lucide:user-round' : 'lucide:gamepad-2'" />
        <template v-if="mine">
          <span v-if="myStatusLabel">{{ myStatusLabel }}</span>
          <span v-if="myDuration" class="tabular-nums">{{ myDuration }}</span>
        </template>
        <template v-else-if="unknown">我的游玩状态</template>
        <template v-else>标记游玩状态</template>
      </KunButton>
    </KunTooltip>

    <GalgameHeaderPlaytimeModal
      v-if="id"
      v-model="isOpen"
      :galgame="galgame"
      :mine="mine ?? null"
      @saved="(value) => (saved = value)"
      @finished="onFinished"
    />
  </div>
</template>
