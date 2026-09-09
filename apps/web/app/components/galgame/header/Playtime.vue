<script setup lang="ts">
import {
  KUN_GALGAME_PLAYTIME_SOURCE_CONST,
  KUN_GALGAME_PLAYTIME_SOURCE_MAP,
  KUN_GALGAME_PLAY_STATE_MAP,
  type KunGalgamePlayState,
  type KunGalgamePlayStateRead
} from '~/constants/galgame-playtime'

const props = defineProps<{
  galgame: GalgameDetail
}>()

const emits = defineEmits<{
  wantsRating: [KunGalgamePlayState]
}>()

const { id } = usePersistUserStore()

const mine = ref<GalgameMyPlaytime | null>(props.galgame.my_playtime ?? null)
watch(
  () => props.galgame.my_playtime,
  (value) => (mine.value = value ?? null)
)

const isOpen = ref(false)

const chips = computed(() =>
  KUN_GALGAME_PLAYTIME_SOURCE_CONST.flatMap((source) => {
    const row = props.galgame.playtimes?.find((p) => p.source === source)
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
  if (!mine.value?.status) return ''
  return (
    KUN_GALGAME_PLAY_STATE_MAP[mine.value.status as KunGalgamePlayStateRead] ??
    mine.value.status
  )
})

const myTooltip = computed(() => {
  if (!mine.value) return '标记你在这部作品上的游玩状态, 也可以记下用时'
  const parts: string[] = []
  if (myStatusLabel.value && myDuration.value) {
    parts.push(`你的记录: ${myDuration.value} · ${myStatusLabel.value}`)
  } else if (myStatusLabel.value) {
    parts.push(`你的记录: ${myStatusLabel.value}`)
  } else if (myDuration.value) {
    parts.push(`你的记录: ${myDuration.value}`)
  }
  const site = props.galgame.playtimes?.find((p) => p.source === 'nextmoe')
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
  if (props.galgame.ratings.some((r) => r.user.id === id)) return
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
        @click="openEditor"
      >
        <KunIcon :name="mine ? 'lucide:user-round' : 'lucide:gamepad-2'" />
        <template v-if="mine">
          <span v-if="myStatusLabel">{{ myStatusLabel }}</span>
          <span v-if="myDuration" class="tabular-nums">{{ myDuration }}</span>
        </template>
        <template v-else>标记游玩状态</template>
      </KunButton>
    </KunTooltip>

    <GalgameHeaderPlaytimeModal
      v-if="id"
      v-model="isOpen"
      :galgame="galgame"
      :mine="mine"
      @saved="(value) => (mine = value)"
      @finished="onFinished"
    />
  </div>
</template>
