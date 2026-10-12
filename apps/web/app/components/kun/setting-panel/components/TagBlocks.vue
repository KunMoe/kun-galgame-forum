<script setup lang="ts">
import { watchDebounced } from '@vueuse/core'
import { settle } from '#shared/utils/api/problem'
import type { TagSummary } from '#shared/utils/api/schemas'
import { catalogVocabularyName } from '#shared/utils/catalogName'
import {
  KUN_TAG_BLOCKS_MAX,
  type KunTagBlockMode
} from '~/store/modules/tagBlocks'

const api = useApiClient()
const { isSignedIn, blocks, count, modeOf, block, unblock, refresh } =
  useTagBlocks()

const summaries = reactive(new Map<number, TagSummary>())
const remember = (row: TagSummary) => summaries.set(Number(row.id), row)
const nameOf = (id: number) => {
  const row = summaries.get(id)
  return row ? catalogVocabularyName(row) : `#${id}`
}
const descendantsOf = (id: number) => summaries.get(id)?.descendant_count ?? 0

const blockedIds = computed(() =>
  [...blocks.value.tree, ...blocks.value.exact].sort((a, b) =>
    nameOf(a).localeCompare(nameOf(b), 'zh-Hans-CN')
  )
)

const resolve = async (ids: number[]) => {
  const missing = ids.filter((id) => !summaries.has(id))
  if (!missing.length) {
    return
  }
  const result = await settle(
    api.GET('/tags', {
      params: {
        query: {
          ids: missing.map(String),
          limit: missing.length,
          include_nsfw: true
        }
      }
    })
  )
  if (result.ok) {
    result.data.items.forEach(remember)
  }
}

watch(blockedIds, resolve, { immediate: true })

onMounted(() => {
  if (isSignedIn.value) {
    refresh()
  }
})

const keyword = ref('')
const results = ref<TagSummary[]>([])
const searching = ref(false)
const confirming = ref<number | null>(null)

const search = async () => {
  const q = keyword.value.trim()
  confirming.value = null
  if (!q) {
    results.value = []
    return
  }
  searching.value = true
  const result = await settle(
    api.GET('/tags', {
      params: { query: { q, limit: 20, include_nsfw: true } }
    })
  )
  searching.value = false
  results.value = result.ok ? result.data.items : []
  results.value.forEach(remember)
}

watchDebounced(keyword, search, { debounce: 400, maxWait: 1000 })

const busy = ref<number | null>(null)
const expanded = ref<number | null>(null)

const run = async (
  id: number,
  action: () => Promise<boolean>,
  done: string
) => {
  busy.value = id
  const ok = await action()
  busy.value = null
  if (ok) {
    useMessage(done, 'success')
  }
  return ok
}

const onBlock = async (id: number, mode: KunTagBlockMode) => {
  const n = descendantsOf(id)
  const done =
    mode === 'tree' && n
      ? `已屏蔽「${nameOf(id)}」及其 ${n} 个子标签`
      : `已屏蔽「${nameOf(id)}」`
  if (await run(id, () => block(id, mode), done)) {
    confirming.value = null
  }
}

const onPick = (id: number) => {
  if (!descendantsOf(id)) {
    onBlock(id, 'tree')
    return
  }
  confirming.value = confirming.value === id ? null : id
}

const onUnblock = (id: number) =>
  run(id, () => unblock(id), `已取消屏蔽「${nameOf(id)}」`)

const onMode = (id: number, mode: KunTagBlockMode) => {
  if (modeOf(id) !== mode) {
    run(
      id,
      () => block(id, mode),
      mode === 'tree'
        ? `「${nameOf(id)}」改为连同子标签一起屏蔽`
        : `「${nameOf(id)}」改为只屏蔽这一个标签`
    )
  }
}

const modeOptions = (id: number) => [
  { value: 'tree' as const, label: `含 ${descendantsOf(id)} 个子标签` },
  { value: 'exact' as const, label: '仅此标签' }
]
</script>

<template>
  <div class="space-y-4">
    <div class="space-y-1">
      <div class="flex items-center justify-between gap-3">
        <p class="text-default-700 font-medium">屏蔽标签</p>
        <KunChip
          size="xs"
          variant="flat"
          :color="count >= KUN_TAG_BLOCKS_MAX ? 'danger' : 'default'"
        >
          <span class="tabular-nums"
            >{{ count }} / {{ KUN_TAG_BLOCKS_MAX }}</span
          >
        </KunChip>
      </div>
      <p class="text-default-500 text-sm">
        带有被屏蔽标签的 Galgame 不会出现在列表、搜索、发售日历和排行中。
        <template v-if="isSignedIn">设置跟随您的 NextMoe 账号。</template>
        <template v-else>未登录时只保存在当前浏览器。</template>
      </p>
    </div>

    <div class="space-y-2">
      <KunInput
        v-model="keyword"
        type="text"
        size="sm"
        placeholder="搜索要屏蔽的标签, 例如 NTR"
        is-clearable
      >
        <template #prefix>
          <KunIcon name="lucide:search" class="text-default-400 size-4" />
        </template>
      </KunInput>

      <p
        v-if="searching"
        class="text-default-400 flex items-center gap-1.5 text-sm"
      >
        <KunIcon name="lucide:loader-circle" class="size-4 animate-spin" />
        正在搜索
      </p>

      <p
        v-else-if="keyword.trim() && !results.length"
        class="text-default-400 text-sm"
      >
        没有找到「{{ keyword.trim() }}」
      </p>

      <KunScrollShadow
        v-else-if="results.length"
        axis="vertical"
        scrollbar="thin"
        class-name="max-h-80"
        content-class="gap-0"
      >
        <div
          v-for="tag in results"
          :key="tag.id"
          class="border-default-100 border-b py-2 last:border-b-0"
        >
          <div class="flex items-center justify-between gap-3">
            <div class="min-w-0">
              <p
                :class="
                  cn(
                    'truncate text-sm',
                    tag.is_sexual ? 'text-danger-600' : 'text-default-700'
                  )
                "
              >
                {{ catalogVocabularyName(tag) }}
              </p>
              <p class="text-default-400 text-xs">
                {{
                  tag.descendant_count
                    ? `含 ${tag.descendant_count} 个子标签`
                    : '没有子标签'
                }}
              </p>
            </div>

            <KunChip
              v-if="modeOf(Number(tag.id))"
              size="xs"
              variant="flat"
              color="danger"
            >
              已屏蔽
            </KunChip>
            <KunButton
              v-else
              size="xs"
              variant="flat"
              color="danger"
              :disabled="busy === Number(tag.id)"
              @click="onPick(Number(tag.id))"
            >
              屏蔽
              <KunIcon
                v-if="tag.descendant_count"
                :name="
                  confirming === Number(tag.id)
                    ? 'lucide:chevron-up'
                    : 'lucide:chevron-down'
                "
                class="size-3"
              />
            </KunButton>
          </div>

          <div
            v-if="confirming === Number(tag.id)"
            class="bg-default-50 mt-2 space-y-3 rounded-lg p-3"
          >
            <GalgameTagBlockCoverage
              :tag-id="Number(tag.id)"
              :count="tag.descendant_count"
            />
            <div class="flex flex-wrap gap-2">
              <KunButton
                size="xs"
                color="danger"
                :disabled="busy === Number(tag.id)"
                @click="onBlock(Number(tag.id), 'tree')"
              >
                连同子标签一起屏蔽
              </KunButton>
              <KunButton
                size="xs"
                variant="flat"
                :disabled="busy === Number(tag.id)"
                @click="onBlock(Number(tag.id), 'exact')"
              >
                只屏蔽这一个
              </KunButton>
            </div>
          </div>
        </div>
      </KunScrollShadow>
    </div>

    <div class="space-y-2">
      <p class="text-default-600 text-sm font-medium">已屏蔽的标签</p>

      <p v-if="!blockedIds.length" class="text-default-400 text-sm">
        还没有屏蔽任何标签。
      </p>

      <div
        v-for="id in blockedIds"
        :key="id"
        class="border-default-200 space-y-2 rounded-lg border px-3 py-2"
      >
        <div class="flex items-center justify-between gap-3">
          <KunLink
            :to="`/galgame/tag/${id}`"
            underline="none"
            :color="summaries.get(id)?.is_sexual ? 'danger' : 'default'"
            class="truncate text-sm font-medium"
          >
            {{ nameOf(id) }}
          </KunLink>
          <KunTooltip text="取消屏蔽" position="left">
            <KunButton
              size="xs"
              variant="light"
              :is-icon-only="true"
              :disabled="busy === id"
              aria-label="取消屏蔽"
              @click="onUnblock(id)"
            >
              <KunIcon name="lucide:x" />
            </KunButton>
          </KunTooltip>
        </div>

        <div
          v-if="descendantsOf(id)"
          class="flex flex-wrap items-center justify-between gap-2"
        >
          <KunRadioGroup
            :model-value="modeOf(id) ?? 'tree'"
            :options="modeOptions(id)"
            variant="pill"
            orientation="horizontal"
            size="xs"
            :disabled="busy === id"
            :aria-label="`${nameOf(id)} 的屏蔽范围`"
            @update:model-value="(mode: KunTagBlockMode) => onMode(id, mode)"
          />
          <KunButton
            size="xs"
            variant="light"
            @click="expanded = expanded === id ? null : id"
          >
            {{ expanded === id ? '收起' : '查看子标签' }}
            <KunIcon
              :name="
                expanded === id ? 'lucide:chevron-up' : 'lucide:chevron-down'
              "
              class="size-3"
            />
          </KunButton>
        </div>
        <p v-else class="text-default-400 text-xs">没有子标签</p>

        <GalgameTagBlockDescendants v-if="expanded === id" :tag-id="id" />
      </div>
    </div>
  </div>
</template>
