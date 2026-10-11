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

const names = reactive(new Map<number, string>())
const rememberTag = (row: TagSummary) =>
  names.set(Number(row.id), catalogVocabularyName(row))
const nameOf = (id: number) => names.get(id) ?? `#${id}`

const blockedIds = computed(() =>
  [...blocks.value.tree, ...blocks.value.exact].sort((a, b) => a - b)
)

const resolveNames = async (ids: number[]) => {
  const missing = ids.filter((id) => !names.has(id))
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
    result.data.items.forEach((row) => rememberTag(row as TagSummary))
  }
}

watch(blockedIds, (ids) => resolveNames(ids), { immediate: true })

onMounted(() => {
  if (isSignedIn.value) {
    refresh()
  }
})

const keyword = ref('')
const results = ref<TagSummary[]>([])
const searching = ref(false)

const search = async () => {
  const q = keyword.value.trim()
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
  results.value = result.ok ? (result.data.items as TagSummary[]) : []
  results.value.forEach(rememberTag)
}

watchDebounced(keyword, search, { debounce: 400, maxWait: 1000 })

const busy = ref<number | null>(null)

const run = async (id: number, action: () => Promise<boolean>) => {
  busy.value = id
  await action()
  busy.value = null
}

const onBlock = (id: number) => run(id, () => block(id, 'tree'))
const onUnblock = (id: number) => run(id, () => unblock(id))
const onMode = (id: number, includeChildren: boolean) => {
  const mode: KunTagBlockMode = includeChildren ? 'tree' : 'exact'
  if (modeOf(id) !== mode) {
    run(id, () => block(id, mode))
  }
}
</script>

<template>
  <div class="space-y-3">
    <div class="space-y-0.5">
      <p class="text-default-700 font-medium">屏蔽标签</p>
      <p class="text-default-500 text-sm">
        带有被屏蔽标签的 Galgame 不会出现在列表、搜索、发售日历和排行中。
        「含子标签」会连同该标签下更细分的标签一起屏蔽，关闭则只屏蔽这一个标签。
        <template v-if="isSignedIn">
          这项设置跟着您的 NextMoe 账号走。
        </template>
        <template v-else>未登录时只保存在当前浏览器。</template>
      </p>
    </div>

    <KunInput v-model="keyword" type="text" placeholder="搜索要屏蔽的标签" />

    <KunLoading :loading="searching">
      <div v-if="results.length" class="flex flex-wrap gap-2">
        <KunButton
          v-for="tag in results"
          :key="tag.id"
          size="sm"
          variant="flat"
          :color="modeOf(Number(tag.id)) ? 'default' : 'danger'"
          :disabled="!!modeOf(Number(tag.id)) || busy === Number(tag.id)"
          @click="onBlock(Number(tag.id))"
        >
          <KunIcon name="lucide:ban" />
          {{ catalogVocabularyName(tag) }}
        </KunButton>
      </div>
    </KunLoading>

    <p class="text-default-500 text-sm">
      已屏蔽 {{ count }} / {{ KUN_TAG_BLOCKS_MAX }}
    </p>

    <div v-if="blockedIds.length" class="space-y-2">
      <div
        v-for="id in blockedIds"
        :key="id"
        class="border-default-200 flex items-center justify-between gap-3 rounded-lg border px-3 py-2"
      >
        <KunLink :to="`/galgame/tag/${id}`" color="default" class="truncate">
          {{ nameOf(id) }}
        </KunLink>
        <div class="flex shrink-0 items-center gap-3">
          <span class="text-default-500 text-sm">含子标签</span>
          <KunSwitch
            :model-value="modeOf(id) === 'tree'"
            :disabled="busy === id"
            @update:model-value="(value: boolean) => onMode(id, value)"
          />
          <KunButton
            size="sm"
            variant="light"
            :is-icon-only="true"
            :disabled="busy === id"
            aria-label="取消屏蔽"
            @click="onUnblock(id)"
          >
            <KunIcon name="lucide:x" />
          </KunButton>
        </div>
      </div>
    </div>
  </div>
</template>
