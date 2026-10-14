<script setup lang="ts">
import type { UserWorkRecord } from '#shared/utils/api/schemas'
import {
  KUN_GALGAME_PLAY_STATE_MAP,
  kunPlayStateColor,
  type KunGalgamePlayStateRead
} from '~/constants/galgame-playtime'

const props = defineProps<{
  record: UserWorkRecord
}>()

const state = computed(() => props.record.play_state ?? '')
const stateLabel = computed(
  () =>
    KUN_GALGAME_PLAY_STATE_MAP[state.value as KunGalgamePlayStateRead] ??
    state.value
)
const duration = computed(() =>
  formatDurationMinutes(props.record.playtime_minutes ?? 0)
)
</script>

<template>
  <div class="mt-1.5 flex flex-wrap items-center gap-1.5 text-xs">
    <KunChip
      v-if="state"
      size="xs"
      variant="flat"
      :color="kunPlayStateColor(state)"
    >
      {{ stateLabel }}
    </KunChip>
    <span v-if="duration" class="text-default-500 tabular-nums">
      {{ duration }}
    </span>
    <span v-if="record.user_rating" class="text-warning-text tabular-nums">
      评分 {{ record.user_rating.overall }}
    </span>
  </div>
</template>
