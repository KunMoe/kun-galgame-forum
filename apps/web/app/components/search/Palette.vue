<script setup lang="ts">
import type { KunCommandGroup, KunCommandItem } from '@kungal/ui-vue'
import { watchDebounced } from '@vueuse/core'
import { KUN_TOPIC_SECTION } from '~/constants/topic'
import { settle } from '#shared/utils/api/problem'
import type {
  TopicSummary,
  UserSearchHit,
  WorkRef
} from '#shared/utils/api/schemas'
import { catalogNameText } from '~/utils/catalogName'

// Mirrors the API's `max=107` on keywords: past it the request is a validation
// error, and one per keystroke would be an error toast per keystroke.
const MAX_KEYWORDS_LENGTH = 107

const open = ref(false)
const query = ref('')
const pending = ref(false)
interface QuickResult {
  topics: TopicSummary[]
  works: WorkRef[]
  users: UserSearchHit[]
  totals: { topic?: number; work?: number; user?: number }
}

const QUICK_LIMIT = 5

const result = ref<QuickResult | null>(null)
const api = useApiClient()
const { allowsNsfw } = useContentStance()
const { showKUNGalgamePreferOriginalName } = storeToRefs(
  usePersistSettingsStore()
)

const { searchHistory } = storeToRefs(usePersistKUNGalgameSearchStore())

const keywords = computed(() => query.value.trim())

let latest = 0

const search = async (value: string) => {
  const current = ++latest
  if (!value || value.length > MAX_KEYWORDS_LENGTH) {
    pending.value = false
    result.value = null
    return
  }

  pending.value = true
  const lane = { q: value, limit: QUICK_LIMIT }
  const [topics, works, users] = await Promise.all([
    settle(
      api.GET('/search/topics', {
        params: { query: { ...lane, include_nsfw: allowsNsfw.value } }
      })
    ),
    settle(
      api.GET('/search/works', {
        params: { query: { ...lane, include_nsfw: allowsNsfw.value } }
      })
    ),
    settle(api.GET('/search/users', { params: { query: lane } }))
  ])
  if (current !== latest) {
    return
  }
  pending.value = false
  result.value = {
    topics: topics.ok ? topics.data.items : [],
    works: works.ok ? works.data.items : [],
    users: users.ok ? users.data.items : [],
    totals: {
      topic: topics.ok ? topics.data.total : undefined,
      work: works.ok ? works.data.total : undefined,
      user: users.ok ? users.data.total : undefined
    }
  }
}

watchDebounced(keywords, search, { debounce: 300 })

watch(open, (isOpen) => {
  if (!isOpen) {
    latest++
    pending.value = false
    result.value = null
  }
})

const hasHit = computed(() => {
  const hit = result.value
  return !!hit && hit.topics.length + hit.works.length + hit.users.length > 0
})

// The action row is always present, so the palette's own no-result text can
// never show. Say it here instead, and say what the search page adds: quick
// search covers three of its six tabs.
const actionDescription = computed(() =>
  !pending.value && result.value && !hasHit.value
    ? '快速搜索没有匹配, 搜索页面还会搜索 Gal 工具资源, 回复与评论'
    : '在搜索页面查看话题, Galgame, 用户, 回复与评论的全部结果'
)

const label = (name: string, total?: number) =>
  total ? `${name} · ${total}` : name

const groups = computed<KunCommandGroup[]>(() => {
  if (!keywords.value) {
    if (!searchHistory.value.length) {
      return []
    }
    return [
      {
        label: '搜索历史',
        items: searchHistory.value
          .slice()
          .reverse()
          .map((history) => ({
            value: `q:${history}`,
            label: history,
            icon: 'lucide:history'
          }))
      }
    ]
  }

  const list: KunCommandGroup[] = [
    {
      items: [
        {
          value: `q:${keywords.value}`,
          label: `搜索「${keywords.value}」`,
          description: actionDescription.value,
          icon: 'lucide:search'
        }
      ]
    }
  ]
  if (!result.value) {
    return list
  }

  const { topics, works, users, totals } = result.value
  if (topics.length) {
    list.push({
      label: label('话题', totals.topic),
      items: topics.map((topic) => ({
        value: `/topic/${topic.id}`,
        label: topic.title,
        description: topic.sections
          .map((name) => KUN_TOPIC_SECTION[name] ?? name)
          .join(' · '),
        icon: 'lucide:message-square-text'
      }))
    })
  }
  if (works.length) {
    list.push({
      label: label('Galgame', totals.work),
      items: works.map((work) => {
        const name = catalogNameText(
          work,
          showKUNGalgamePreferOriginalName.value
        )
        return {
          value: `/galgame/${work.id}`,
          label: name,
          description:
            name === work.display_name ? (work.latin ?? '') : work.display_name,
          icon: 'lucide:gamepad-2'
        }
      })
    })
  }
  if (users.length) {
    list.push({
      label: label('用户', totals.user),
      items: users.map((user) => ({
        value: `/user/${user.id}`,
        label: user.name ?? '',
        description: user.bio ?? '',
        icon: 'lucide:user-round'
      }))
    })
  }
  return list
})

const rememberHistory = (value: string) => {
  const index = searchHistory.value.indexOf(value)
  if (index !== -1) {
    searchHistory.value.splice(index, 1)
  }
  searchHistory.value.push(value)
  searchHistory.value = searchHistory.value.slice(-20)
}

// The palette never navigates while typing. The first row is always the search
// action, so a bare Enter goes to the search page and only an explicitly moved
// selection opens a single result.
const handleSelect = (item: KunCommandItem) => {
  const value = String(item.value)
  if (!value.startsWith('q:')) {
    navigateTo(value)
    return
  }

  const typed = value.slice(2)
  rememberHistory(typed)
  navigateTo({ path: '/search', query: { keywords: typed } })
}
</script>

<template>
  <KunCommandPalette
    v-model:open="open"
    v-model:query="query"
    :items="groups"
    :loading="pending && !hasHit"
    placeholder="搜索话题, Galgame, 用户…"
    empty-text="输入关键字以搜索整个论坛"
    aria-label="站内搜索"
    @select="handleSelect"
  >
    <template #trigger="{ open: openPalette, shortcut }">
      <KunTooltip :text="`${shortcut} 以快速搜索`" position="bottom">
        <KunButton
          :is-icon-only="true"
          variant="light"
          color="default"
          size="xl"
          @click="openPalette"
        >
          <KunIcon name="lucide:search" />
        </KunButton>
      </KunTooltip>
    </template>
  </KunCommandPalette>
</template>
