<script setup lang="ts">
import {
  KUN_GALGAME_PLAYTIME_SOURCE_CONST,
  KUN_GALGAME_PLAYTIME_SOURCE_MAP,
  KUN_GALGAME_PLAYTIME_STATUS_MAP,
  type KunGalgameWorkState
} from '~/constants/galgame-playtime'

const props = defineProps<{
  galgame: GalgameDetail
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

const myDuration = computed(() => formatDurationMinutes(mine.value?.minutes))

const myTooltip = computed(() => {
  if (!mine.value) return '记录你在这部作品上的游玩时长'
  const status =
    KUN_GALGAME_PLAYTIME_STATUS_MAP[mine.value.status as KunGalgameWorkState] ??
    ''
  const parts = [
    status
      ? `你的记录: ${myDuration.value} · ${status}`
      : `你的记录: ${myDuration.value}`
  ]
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
        <KunIcon :name="mine ? 'lucide:user-round' : 'lucide:timer'" />
        <template v-if="mine">
          <span class="tabular-nums">{{ myDuration }}</span>
        </template>
        <template v-else>记录我的时长</template>
      </KunButton>
    </KunTooltip>

    <GalgameHeaderPlaytimeModal
      v-if="id"
      v-model="isOpen"
      :galgame="galgame"
      :mine="mine"
      @saved="(value) => (mine = value)"
    />
  </div>
</template>
