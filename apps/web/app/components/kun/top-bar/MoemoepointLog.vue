<script setup lang="ts">
interface MoemoepointLogEntry {
  id: number
  delta: number
  reason: string
  source_app: string
  ref: string
  created_at: string
  is_local: boolean
}

const PAGE_SIZE = 20

const { showKUNGalgameMoemoepointLog: isOpen } = storeToRefs(
  useTempSettingStore()
)
const { moemoepoint } = storeToRefs(usePersistUserStore())

const REASON_META: Record<string, { label: string; icon: string }> = {
  daily_checkin: { label: '每日签到', icon: 'lucide:calendar-check' },
  liked: { label: '内容被点赞', icon: 'lucide:heart' },
  content_approved: { label: '内容被采纳', icon: 'lucide:circle-check-big' },
  content_removed: { label: '内容被移除', icon: 'lucide:circle-x' },
  admin_grant: { label: '管理员发放', icon: 'lucide:gift' },
  admin_deduct: { label: '管理员扣除', icon: 'lucide:gavel' },
  migration: { label: '初始迁移', icon: 'lucide:database' },
  register_gift: { label: '注册礼物', icon: 'lucide:party-popper' },
  name_change: { label: '修改用户名', icon: 'lucide:user-round-pen' }
}

const SOURCE_LABEL: Record<string, string> = {
  kungal: '鲲 Galgame',
  moyu: '鲲补丁',
  patch: '鲲补丁',
  touchgal: 'TouchGal',
  sticker: '贴纸小铺',
  stickers: '贴纸小铺',
  oauth: '账号中心'
}

const REF_KIND_LABEL: Record<string, string> = {
  galgame: 'Galgame',
  galgame_pr: 'Galgame 修订',
  galgame_comment: '游戏评论',
  galgame_rating: '游戏评分',
  galgame_resource: '游戏资源',
  galgame_quiz: '题目',
  toolset: '工具集',
  toolset_resource: '工具资源',
  topic: '话题',
  topic_comment: '话题评论',
  topic_reply: '回复'
}

const reasonMeta = (reason: string) =>
  REASON_META[reason] ?? {
    label: reason || '萌萌点变动',
    icon: 'lucide:lollipop'
  }

const REASON_ACTION: Record<string, { pos: string; neg: string }> = {
  liked: { pos: '被点赞', neg: '取消点赞' },
  content_removed: { pos: '被移除', neg: '被移除' }
}

const BEHAVIOR_LABEL: Record<string, string> = {
  'content_approved:galgame_quiz_answer': '答对了题目',
  'content_approved:topic_upvote': '话题被推荐',
  'content_removed:topic_upvote': '推话题消耗'
}

const refKindOf = (ref: string) => ref.split(':')[0] ?? ''

const behaviorLabel = (entry: MoemoepointLogEntry): string => {
  const kind = refKindOf(entry.ref)
  const specific = BEHAVIOR_LABEL[`${entry.reason}:${kind}`]
  if (specific) return specific
  const kindLabel = REF_KIND_LABEL[kind]
  if (kindLabel) {
    if (entry.reason === 'content_approved') {
      return entry.delta < 0 ? `${kindLabel}被移除` : `创建了新的${kindLabel}`
    }
    const action = REASON_ACTION[entry.reason]
    if (action)
      return `${kindLabel}${entry.delta < 0 ? action.neg : action.pos}`
  }
  return reasonMeta(entry.reason).label
}

const REF_LINK_BASE: Record<string, string> = {
  topic: '/topic',
  topic_upvote: '/topic',
  galgame: '/galgame',
  galgame_pr: '/galgame',
  galgame_quiz: '/galgame-quiz',
  toolset: '/toolset'
}

const refHref = (entry: MoemoepointLogEntry): string => {
  if (!entry.is_local) return ''
  const base = REF_LINK_BASE[refKindOf(entry.ref)]
  const id = entry.ref.split(':')[1]
  return base && id ? `${base}/${id}` : ''
}

const isOpaqueId = (value: string) => /^[0-9a-f]{16,}$/i.test(value)

const sourceLabel = (app: string): string => {
  if (!app) return ''
  const slug = app.replace(/-backend$/, '')
  if (SOURCE_LABEL[slug]) return SOURCE_LABEL[slug]
  return isOpaqueId(slug) ? '' : slug
}

const refId = (refValue: string): string => {
  const id = refValue.split(':')[1]
  return id ? `#${id}` : ''
}

const metaSegments = (
  entry: MoemoepointLogEntry
): { text: string; href?: string }[] => {
  const segments: { text: string; href?: string }[] = []
  const source = sourceLabel(entry.source_app)
  if (source) segments.push({ text: source })
  const id = refId(entry.ref)
  if (id) segments.push({ text: id, href: refHref(entry) || undefined })
  segments.push({ text: formatTimeDifference(entry.created_at) })
  return segments
}

const entries = ref<MoemoepointLogEntry[]>([])
const status = ref<'idle' | 'loading' | 'loadingMore' | 'error'>('idle')
const hasMore = ref(true)

const fetchPage = async (more = false) => {
  if (more && (!hasMore.value || status.value === 'loadingMore')) return
  status.value = more ? 'loadingMore' : 'loading'

  const beforeId =
    more && entries.value.length
      ? entries.value[entries.value.length - 1]!.id
      : 0

  const page = await kunFetch<{
    items: MoemoepointLogEntry[]
    has_more: boolean
  }>('/user/moemoepoint/log', {
    query: { limit: PAGE_SIZE, before_id: beforeId }
  })

  if (page === null) {
    status.value = 'error'
    return
  }

  entries.value = more ? [...entries.value, ...page.items] : page.items
  hasMore.value = page.has_more
  status.value = 'idle'
}

watch(isOpen, (open) => {
  if (!open) return
  entries.value = []
  hasMore.value = true
  fetchPage(false)
})
</script>

<template>
  <KunModal v-model="isOpen" inner-class-name="max-w-lg w-full">
    <div class="flex max-h-[75dvh] flex-col gap-3 p-1">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <KunIcon class="text-secondary text-2xl" name="lucide:lollipop" />
          <span class="text-lg font-bold">萌萌点明细</span>
        </div>
        <p class="flex items-center gap-1 font-bold">
          <span class="text-default-500 text-sm font-normal">当前</span>
          <span class="text-secondary">{{ moemoepoint }}</span>
        </p>
      </div>

      <p class="text-default-500 text-xs">
        这里汇总了你在鲲 Galgame 全站(及关联站点)的萌萌点收支记录
      </p>

      <KunLoading v-if="status === 'loading'" />

      <KunNull
        v-else-if="status === 'error'"
        description="加载失败, 请稍后再试"
      />

      <KunNull v-else-if="!entries.length" description="暂无萌萌点记录" />

      <div v-else class="flex min-h-0 flex-1 flex-col gap-1 overflow-y-auto">
        <div
          v-for="entry in entries"
          :key="entry.id"
          class="hover:bg-default-100 flex items-center gap-3 rounded-lg p-2 transition-colors"
        >
          <KunIcon
            class="text-default-500 shrink-0 text-xl"
            :name="reasonMeta(entry.reason).icon"
          />
          <div class="flex min-w-0 grow flex-col">
            <span class="truncate text-sm font-medium">
              {{ behaviorLabel(entry) }}
            </span>
            <span
              class="text-default-400 flex items-center gap-1 truncate text-xs"
            >
              <template v-for="(seg, i) in metaSegments(entry)" :key="i">
                <span v-if="i > 0">·</span>
                <KunLink
                  v-if="seg.href"
                  :to="seg.href"
                  class="hover:text-primary cursor-pointer transition-colors"
                  @click="isOpen = false"
                >
                  {{ seg.text }}
                </KunLink>
                <span v-else>{{ seg.text }}</span>
              </template>
            </span>
          </div>
          <span
            class="shrink-0 text-sm font-bold tabular-nums"
            :class="entry.delta >= 0 ? 'text-success-600' : 'text-danger-500'"
          >
            {{ entry.delta >= 0 ? '+' : '' }}{{ entry.delta }}
          </span>
        </div>

        <KunButton
          v-if="hasMore"
          variant="light"
          class-name="mt-1"
          :loading="status === 'loadingMore'"
          @click="fetchPage(true)"
        >
          加载更多
        </KunButton>
      </div>
    </div>
  </KunModal>
</template>
