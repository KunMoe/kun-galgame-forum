<script setup lang="ts">
import type { KunDropdownItem } from '@kungal/ui-vue'
import { settle } from '#shared/utils/api/problem'
import type { WorkViewerPlaytime } from '#shared/utils/api/schemas'
import {
  KUN_GALGAME_PLAY_STATE_BASE,
  kunPlayStateBase,
  type KunGalgamePlayStateBase
} from '~/constants/galgame-playtime'

const props = withDefaults(
  defineProps<{
    workId: string | number
    playState?: string | null
    size?: 'xs' | 'sm'
  }>(),
  { playState: undefined, size: 'sm' }
)

const emits = defineEmits<{
  changed: [playtime: WorkViewerPlaytime, ratingDeleted: boolean]
}>()

const { id } = usePersistUserStore()
const api = useApiClient()
const { playtimeOf, setPlaytime, ensureLoaded } = useMyGalgameInteractions()
onMounted(() => {
  if (props.playState === undefined) {
    ensureLoaded([props.workId])
  }
})

const current = computed(() =>
  props.playState === undefined
    ? (playtimeOf(props.workId)?.play_state ?? null)
    : props.playState
)
const active = computed(() => kunPlayStateBase(current.value))
const activeLabel = computed(
  () =>
    KUN_GALGAME_PLAY_STATE_BASE.find((s) => s.value === active.value)?.label ??
    ''
)

const items = computed<KunDropdownItem[]>(() => [
  ...KUN_GALGAME_PLAY_STATE_BASE.map((s) => ({
    key: s.value,
    label: s.label,
    icon: s.value === active.value ? 'lucide:check' : s.icon,
    color: s.value === active.value ? ('primary' as const) : undefined
  })),
  ...(active.value
    ? [
        {
          key: 'clear',
          label: '取消标记',
          icon: 'lucide:circle-x',
          color: 'danger' as const
        }
      ]
    : [])
])

const pending = ref(false)

// The forum keeps a wish unrated; catalog does not enforce it, so the rating
// goes first and only then the state.
const dropRatingForWish = async (): Promise<boolean | null> => {
  const record = await settle(
    api.GET('/works/{work_id}/record', {
      params: { path: { work_id: String(props.workId) } }
    })
  )
  if (!record.ok) {
    reportProblem(record.problem)
    return null
  }
  const rating = record.data.rating_summary
  if (!rating) return false
  const ok = await useComponentMessageStore().alert(
    '标记为想玩会删除你的评分',
    '想玩的作品不能有评分。评分在所有 NextMoe 站点共用, 删除后其它站点上也会一起消失。'
  )
  if (!ok) return null
  const deleted = await settle(
    api.DELETE('/ratings/{rating_id}', {
      params: { path: { rating_id: rating.id } }
    })
  )
  if (!deleted.ok) {
    reportProblem(deleted.problem)
    return null
  }
  return true
}

const choose = async (item: KunDropdownItem) => {
  if (pending.value) return
  if (!id) {
    useAuthModal().open()
    return
  }
  const next =
    item.key === 'clear' ? null : (item.key as KunGalgamePlayStateBase)
  if (next === active.value) return

  pending.value = true
  let ratingDeleted = false
  if (next === 'wish') {
    const dropped = await dropRatingForWish()
    if (dropped === null) {
      pending.value = false
      return
    }
    ratingDeleted = dropped
  }
  const result = await settle(
    api.PUT('/works/{work_id}/playtime', {
      params: { path: { work_id: String(props.workId) } },
      body: { play_state: next }
    })
  )
  pending.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  const saved = result.data
  setPlaytime(
    props.workId,
    saved.play_state || saved.minutes > 0 ? saved : null
  )
  emits('changed', saved, ratingDeleted)
  useMessage(
    next
      ? `已标记为「${KUN_GALGAME_PLAY_STATE_BASE.find((s) => s.value === next)?.label}」`
      : '已取消标记',
    'success'
  )
}
</script>

<template>
  <KunDropdown
    :items="items"
    position="bottom-end"
    :disabled="pending"
    @select="choose"
  >
    <template #trigger>
      <KunButton
        :variant="active ? 'flat' : 'bordered'"
        :color="active ? 'primary' : 'default'"
        :size="size"
        :loading="pending"
        :aria-label="active ? `游玩状态: ${activeLabel}` : '标记游玩状态'"
      >
        <span class="flex items-center gap-1">
          <KunIcon :name="active ? 'lucide:check' : 'lucide:bookmark-plus'" />
          {{ active ? activeLabel : '标记' }}
        </span>
      </KunButton>
    </template>
  </KunDropdown>
</template>
