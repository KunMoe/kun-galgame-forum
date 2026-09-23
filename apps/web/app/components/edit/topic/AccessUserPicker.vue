<script setup lang="ts">
import { watchDebounced } from '@vueuse/core'
import { settle } from '#shared/utils/api/problem'
import { toKunUser } from '~/utils/userRef'

interface TopicAccessUser {
  id: number
  name: string
  avatar: string
}

const props = defineProps<{
  limit: number
  knownUsers?: KunUser[]
}>()

const selected = defineModel<number[]>({ required: true })

const api = useApiClient()
const { id: currentUserId } = usePersistUserStore()
const config = useRuntimeConfig()

const known = ref<Record<number, TopicAccessUser>>({})
const keyword = ref('')
const results = ref<TopicAccessUser[]>([])
const searching = ref(false)

// access_grants.user_ids carries bare ids and user name search only matches names,
// so the floating card is the only id -> name face there is. Raw $fetch on purpose:
// it 404s on a banned or deleted grantee, and kunFetch would pop one error toast
// per unresolvable id the moment the author opens the editor.
const resolve = async (id: number) => {
  const resp = await $fetch<{ code: number; data?: TopicAccessUser }>(
    `${config.public.apiBaseUrl}/api/user/${id}/floating`,
    { credentials: 'include', query: { user_id: id } }
  ).catch(() => null)
  if (resp?.code === 0 && resp.data) {
    known.value[id] = resp.data
  }
}

const resolveMissing = async (ids: number[]) => {
  await Promise.all(ids.filter((id) => !known.value[id]).map(resolve))
}

onMounted(() =>
  watch(selected, (ids) => void resolveMissing(ids), { immediate: true })
)

watch(
  () => props.knownUsers,
  (users) => {
    for (const user of users ?? []) {
      known.value[user.id] = user
    }
  },
  { immediate: true }
)

watchDebounced(
  keyword,
  async (value) => {
    const q = value.trim()
    if (!q) {
      results.value = []
      return
    }
    searching.value = true
    const page = await settle(
      api.GET('/users', {
        params: { query: { q, limit: 8 } }
      })
    )
    searching.value = false
    if (!page.ok) {
      reportProblem(page.problem)
      results.value = []
      return
    }
    results.value = page.data.items
      .map(toKunUser)
      .filter((user) => user.id !== currentUserId)
  },
  { debounce: 300, maxWait: 1000 }
)

const chips = computed(() =>
  selected.value.map((id) => ({
    id,
    name: known.value[id]?.name ?? `用户 #${id}`,
    avatar: known.value[id]?.avatar ?? ''
  }))
)

const add = (user: TopicAccessUser) => {
  keyword.value = ''
  results.value = []
  if (selected.value.includes(user.id)) {
    return
  }
  if (selected.value.length >= props.limit) {
    useMessage(`最多只能指定 ${props.limit} 位用户`, 'warn')
    return
  }
  known.value[user.id] = user
  selected.value = [...selected.value, user.id]
}

const remove = (id: number) => {
  selected.value = selected.value.filter((value) => value !== id)
}
</script>

<template>
  <div class="space-y-2">
    <div class="flex items-center justify-between">
      <span class="text-sm font-medium">指定可以看到本话题的用户</span>
      <span class="text-default-500 text-sm">
        已选 {{ selected.length }}/{{ limit }}
      </span>
    </div>

    <div v-if="chips.length" class="flex flex-wrap gap-2">
      <KunChip
        v-for="chip in chips"
        :key="chip.id"
        color="primary"
        variant="flat"
        size="sm"
        :closable="true"
        @close="remove(chip.id)"
      >
        <template #start>
          <KunAvatar
            :user="{ id: chip.id, name: chip.name, avatar: chip.avatar }"
            size="xs"
            :is-navigation="false"
            :disable-floating="true"
          />
        </template>
        {{ chip.name }}
      </KunChip>
    </div>

    <KunInput
      v-model="keyword"
      placeholder="搜索用户名以添加..."
      aria-label="搜索用户名以添加"
    />

    <p v-if="searching" class="text-default-400 text-sm">搜索中...</p>

    <div
      v-if="results.length"
      class="border-default-200 max-h-48 overflow-y-auto rounded-lg border"
    >
      <button
        v-for="user in results"
        :key="user.id"
        type="button"
        :class="
          cn(
            'hover:bg-default-100 flex w-full items-center gap-2 px-3 py-2',
            'text-left text-sm transition-colors'
          )
        "
        @click="add(user)"
      >
        <KunAvatar
          :user="user"
          size="xs"
          :is-navigation="false"
          :disable-floating="true"
        />
        {{ user.name }}
      </button>
    </div>

    <p v-if="!selected.length" class="text-danger-500 text-sm">
      还没有指定任何用户, 现在只有您自己与管理人员能打开这个话题
    </p>
  </div>
</template>
