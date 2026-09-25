<script setup lang="ts">
import { watchDebounced } from '@vueuse/core'
import { useRouteQuery } from '@vueuse/router'
import type {
  CharacterGender,
  CharacterPage,
  CharacterSort,
  TraitSummary
} from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'

const TRAIT_MAX = 10
const LIMIT = 24
const PAGE_MAX = Math.floor(10000 / LIMIT)

const props = withDefaults(
  defineProps<{
    /** The trait page's own trait: always in the filter, never removable. */
    pinned?: TraitSummary | null
  }>(),
  { pinned: null }
)

const api = useApiClient()
const route = useRoute()
const router = useRouter()
const { allowsNsfw, stanceKey } = useContentStance()

const page = usePageQuery()
const qQuery = useRouteQuery<string>('q', '', { mode: 'replace' })
const traitIdsQuery = useRouteQuery<string>('trait_ids', '', {
  mode: 'replace'
})
const matchQuery = useRouteQuery<string>('trait_match', 'all', {
  mode: 'replace'
})
const sortQuery = useRouteQuery<string>('sort', '', { mode: 'replace' })
const gendersQuery = useRouteQuery<string>('genders', '', { mode: 'replace' })

const pinnedId = computed(() => (props.pinned ? Number(props.pinned.id) : 0))

const pickedIds = computed<number[]>({
  get: () =>
    traitIdsQuery.value
      .split(',')
      .map(Number)
      .filter((id) => Number.isInteger(id) && id > 0 && id !== pinnedId.value),
  set: (ids) => {
    traitIdsQuery.value = ids.join(',')
  }
})
const traitIds = computed(() =>
  pinnedId.value ? [pinnedId.value, ...pickedIds.value] : pickedIds.value
)
const match = computed<'all' | 'any'>(() =>
  matchQuery.value === 'any' ? 'any' : 'all'
)

const GENDERS: CharacterGender[] = ['female', 'male', 'other']
const genders = computed<CharacterGender[]>(() =>
  GENDERS.filter((g) => gendersQuery.value.split(',').includes(g))
)

// The API refuses relevance_desc without q.
const sort = computed<CharacterSort>(() => {
  const picked = sortQuery.value as CharacterSort
  if (picked === 'relevance_desc' || !picked) {
    return qQuery.value ? 'relevance_desc' : 'popularity_desc'
  }
  return picked === 'id_desc' ? 'id_desc' : 'popularity_desc'
})

const picked = reactive(new Map<number, TraitSummary>())

const { data: resolved } = await useAsyncData(
  () => `trait-names:${pickedIds.value.join(',')}:${stanceKey.value}`,
  async () => {
    if (!pickedIds.value.length) {
      return []
    }
    const res = await settle(
      api.GET('/traits', {
        params: {
          query: {
            ids: pickedIds.value.map(String),
            limit: pickedIds.value.length,
            include_nsfw: allowsNsfw.value
          }
        }
      })
    )
    return res.ok ? res.data.items : []
  }
)

const known = computed(() => {
  const map = new Map<number, TraitSummary>()
  for (const t of [...(resolved.value ?? []), ...picked.values()]) {
    map.set(Number(t.id), t)
  }
  if (props.pinned) {
    map.set(pinnedId.value, props.pinned)
  }
  return map
})

const keyword = ref(qQuery.value)
watchDebounced(
  keyword,
  (value) => {
    const q = value.trim()
    if (q !== qQuery.value) {
      qQuery.value = q
      page.value = 1
    }
  },
  { debounce: 400 }
)
watch(qQuery, (q) => {
  if (q !== keyword.value.trim()) {
    keyword.value = q
  }
})

// Both writes land in one navigation: useRouteQuery batches every set made in
// the same tick into a single replace.
const setPicked = (ids: number[]) => {
  pickedIds.value = ids
  page.value = 1
}

const toggle = (t: TraitSummary) => {
  picked.set(Number(t.id), t)
  const id = Number(t.id)
  if (id === pinnedId.value) {
    return
  }
  setPicked(
    pickedIds.value.includes(id)
      ? pickedIds.value.filter((x) => x !== id)
      : [...pickedIds.value, id].slice(0, TRAIT_MAX - (pinnedId.value ? 1 : 0))
  )
}

const setMatch = (value: string | string[]) => {
  matchQuery.value = value === 'any' ? 'any' : 'all'
  page.value = 1
}

const setSort = (value: string | string[]) => {
  sortQuery.value = value as string
  page.value = 1
}

const setGenders = (value: string | string[]) => {
  gendersQuery.value = (value as string[]).join(',')
  page.value = 1
}

const clearAll = () => {
  keyword.value = ''
  qQuery.value = ''
  gendersQuery.value = ''
  setPicked([])
}

const removeChip = (key: string) => {
  if (key === 'q') {
    keyword.value = ''
    qQuery.value = ''
    page.value = 1
    return
  }
  if (key.startsWith('gender:')) {
    setGenders(genders.value.filter((g) => `gender:${g}` !== key))
    return
  }
  setPicked(pickedIds.value.filter((id) => id !== Number(key)))
}

const traitLabel = (id: number) => {
  const t = known.value.get(id)
  return t ? catalogVocabularyName(t) : `#${id}`
}

const GENDER_LABEL: Record<CharacterGender, string> = {
  female: '女性',
  male: '男性',
  other: '其他'
}

const chips = computed<FilterChip[]>(() => [
  ...(qQuery.value ? [{ key: 'q', prefix: '名字', label: qQuery.value }] : []),
  ...genders.value.map((g) => ({
    key: `gender:${g}`,
    prefix: '性别',
    label: GENDER_LABEL[g]
  })),
  ...pickedIds.value.map((id) => {
    const parent = known.value.get(id)?.parents[0]
    return {
      key: String(id),
      prefix: parent ? catalogVocabularyName(parent) : '属性',
      label: traitLabel(id)
    }
  })
])

const genderOptions: FilterOption[] = GENDERS.map((g) => ({
  value: g,
  label: GENDER_LABEL[g]
}))

const sortOptions = computed<FilterOption[]>(() => [
  ...(qQuery.value ? [{ value: 'relevance_desc', label: '名字最匹配' }] : []),
  { value: 'popularity_desc', label: '人气最高', hint: '按登场作品的热度' },
  { value: 'id_desc', label: '最新收录' }
])

const matchOptions: FilterOption[] = [
  { value: 'all', label: '全部满足', hint: '同时拥有所选属性' },
  { value: 'any', label: '满足任一', hint: '拥有任意一个即可' }
]

const query = computed(() => ({
  ...(qQuery.value ? { q: qQuery.value } : {}),
  ...(traitIds.value.length
    ? { trait_ids: traitIds.value.map(String), trait_match: match.value }
    : {}),
  ...(genders.value.length ? { genders: genders.value } : {}),
  sort: sort.value,
  page: page.value,
  limit: LIMIT,
  include_nsfw: allowsNsfw.value
}))

const { data, status, problem } = await useApi<CharacterPage>(
  () => `characters:${JSON.stringify(query.value)}:${stanceKey.value}`,
  (client) => client.GET('/characters', { params: { query: query.value } })
)

const characters = computed(() => data.value?.items ?? [])
const total = computed(() => data.value?.total ?? 0)
const totalPage = computed(() =>
  Math.min(Math.ceil(total.value / LIMIT), PAGE_MAX)
)

const needsNsfw = computed(() =>
  problem.value?.errors.some(
    (e) => e.parameter === 'trait_ids' && e.reason === 'NOT_ALLOWED_VALUE'
  )
)

const emptyText = computed(() => {
  if (traitIds.value.length > 1 && match.value === 'all') {
    return '没有同时满足这些条件的角色, 试试把匹配方式切换为「满足任一」, 或减少几个属性'
  }
  return qQuery.value || traitIds.value.length || genders.value.length
    ? '没有找到符合条件的角色'
    : '暂无角色'
})

const isPanelOpen = ref(
  !props.pinned && !pickedIds.value.length && !qQuery.value
)
const browsedGroup = ref(props.pinned?.trait_group_id ?? '')

const pageHref = (n: number) =>
  router.resolve({ query: { ...route.query, page: n > 1 ? n : undefined } })
    .href
</script>

<template>
  <div class="space-y-4">
    <FilterBar
      :chips="chips"
      :total="data ? total : null"
      :unit="data?.total_relation === 'gte' ? '名以上角色' : '名角色'"
      :pending="status === 'pending'"
      @remove="removeChip"
      @clear="clearAll"
    >
      <div class="w-full sm:w-56">
        <KunInput
          v-model="keyword"
          type="search"
          size="sm"
          rounded="full"
          placeholder="按名字搜索角色"
          aria-label="按名字搜索角色"
        />
      </div>

      <GalgameTraitMenu
        :selected-ids="pickedIds"
        :selected-items="[...known.values()]"
        :max="TRAIT_MAX - (pinnedId ? 1 : 0)"
        @toggle="toggle"
      />

      <FilterMenu
        icon="lucide:venus-and-mars"
        label="性别"
        :options="genderOptions"
        :model-value="genders"
        multiple
        @update:model-value="setGenders"
      />

      <FilterMenu
        icon="lucide:arrow-down-wide-narrow"
        label="排序"
        :options="sortOptions"
        :model-value="sort"
        :empty-value="qQuery ? 'relevance_desc' : 'popularity_desc'"
        @update:model-value="setSort"
      />

      <FilterMenu
        v-if="traitIds.length > 1"
        icon="lucide:git-merge"
        label="匹配方式"
        :options="matchOptions"
        :model-value="match"
        empty-value="all"
        @update:model-value="setMatch"
      />

      <template #end>
        <button
          type="button"
          :class="
            cn(
              'flex shrink-0 cursor-pointer items-center gap-1.5 rounded-full px-3 py-1.5 text-sm',
              filterPillClass(isPanelOpen)
            )
          "
          :aria-expanded="isPanelOpen"
          @click="isPanelOpen = !isPanelOpen"
        >
          <KunIcon name="lucide:list-tree" class="shrink-0" />
          按分类挑选
        </button>
      </template>
    </FilterBar>

    <div v-show="isPanelOpen" class="border-default-200 rounded-xl border p-3">
      <KunScrollShadow axis="vertical" class-name="max-h-96">
        <GalgameTraitGroupBrowser
          v-model:group="browsedGroup"
          :picked-ids="traitIds"
          @pick="toggle"
        />
      </KunScrollShadow>
    </div>

    <KunInfo
      v-if="needsNsfw"
      color="warning"
      title="筛选里有成人向属性"
      description="当前为 SFW 模式, 成人向属性不能用来筛选。请在设置面板开启 NSFW 开关, 或移除这个属性。"
    />

    <KunLoading v-else :loading="status === 'pending'">
      <div
        v-if="characters.length"
        class="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6"
      >
        <GalgameCharacterCard
          v-for="character in characters"
          :key="character.id"
          :character="character"
        />
      </div>

      <KunNull v-else-if="data" :description="emptyText" />

      <KunNull
        v-else-if="problem"
        description="角色列表暂时无法加载, 请稍后重试"
      />
    </KunLoading>

    <p
      v-if="data?.total_relation === 'gte'"
      class="text-default-400 text-center text-xs"
    >
      符合条件的角色超过 {{ total.toLocaleString('en-US') }} 名, 只能翻到前
      {{ PAGE_MAX }} 页。加几个属性或输入名字可以缩小范围。
    </p>

    <KunPagination
      v-if="totalPage > 1"
      v-model:current-page="page"
      :total-page="totalPage"
      :is-loading="status === 'pending'"
      :page-href="pageHref"
    />
  </div>
</template>
