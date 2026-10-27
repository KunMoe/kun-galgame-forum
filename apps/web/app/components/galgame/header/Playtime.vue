<script setup lang="ts">
import {
  KUN_GALGAME_PLAYTIME_SOURCE_CONST,
  KUN_GALGAME_PLAYTIME_SOURCE_MAP
} from '~/constants/galgame-playtime'
import type { Work } from '#shared/utils/api/schemas'

const props = defineProps<{
  galgame: Work
}>()

const { id } = usePersistUserStore()
const { playtimeOf, ensureLoaded } = useMyGalgameInteractions()
onMounted(() => ensureLoaded([props.galgame.id]))

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

const mine = computed(() => (id ? playtimeOf(props.galgame.id) : null))
const myDuration = computed(() =>
  mine.value && mine.value.minutes > 0
    ? formatDurationMinutes(mine.value.minutes)
    : ''
)
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

    <KunSkeleton
      v-if="mine === undefined"
      height="1.5rem"
      width="4.5rem"
      rounded="full"
    />
    <KunTooltip
      v-else-if="myDuration"
      text="你记录的游玩时长"
      class-name="shrink-0"
    >
      <KunChip size="sm" variant="flat" color="primary">
        <KunIcon name="lucide:user-round" />
        <span class="tabular-nums">{{ myDuration }}</span>
      </KunChip>
    </KunTooltip>
    <span v-else-if="!chips.length" class="text-default-400 text-sm">
      未记录
    </span>
  </div>
</template>
