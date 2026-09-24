<script setup lang="ts">
import { watchDebounced } from '@vueuse/core'
import { useRouteQuery } from '@vueuse/router'
import type { PageListGalgameResource } from '#shared/utils/api/schemas'
import { problemMessage } from '#shared/utils/api/message'

const LIMIT = 50

const page = useRouteQuery('page', 1, { mode: 'replace', transform: Number })
const keywords = useRouteQuery<string>('keywords', '', { mode: 'replace' })
const input = ref(keywords.value)
const { allowsNsfw, stanceKey } = useContentStance()

const { data, status, problem } = await useApi<PageListGalgameResource>(
  () =>
    `galgame-resources:${page.value}:${keywords.value}:${stanceKey.value}`,
  (api, { signal }) =>
    api.GET('/galgame-resources', {
      params: {
        query: {
          page: page.value,
          limit: LIMIT,
          include_nsfw: allowsNsfw.value,
          ...(keywords.value ? { q: keywords.value } : {})
        }
      },
      signal
    })
)

watchDebounced(
  input,
  () => {
    const next = input.value.trim()
    if (next === keywords.value) {
      return
    }
    page.value = 1
    keywords.value = next
  },
  { debounce: 500, maxWait: 1000 }
)

watch(page, () => {
  window.scrollTo({ top: 0, behavior: 'smooth' })
})

const resources = computed(() => data.value?.items ?? [])
const total = computed(() => data.value?.total ?? 0)
const isPending = computed(() => status.value === 'pending')

const description =
  '在本页面查看网站所有 Galgame 下载资源列表, 包括 PC / Windows, 手机, 模拟器 / KRKR / Tyranor 等等'

useKunSeoMeta({
  title: '最新 Galgame 资源下载',
  description
})
</script>

<template>
  <div class="space-y-3">
    <KunHeader name="最新 Galgame 资源下载" :description="description">
      <template #endContent>
        <div>
          <KunInput
            v-model="input"
            type="text"
            placeholder="搜索资源备注或 Galgame 名称..."
          />

          <div class="text-default-600 mt-4 text-sm">
            <span v-if="keywords">{{ `搜索结果: ${total} 个资源` }}</span>
            <span v-else>{{ `总计 ${total} 个资源` }}</span>
          </div>
        </div>
      </template>
    </KunHeader>

    <KunNull v-if="problem" :description="problemMessage(problem)" />

    <KunLoading v-else :loading="isPending">
      <div
        v-if="resources.length"
        class="grid grid-cols-1 gap-3 md:grid-cols-2"
      >
        <GalgameResourceCard
          v-for="resource in resources"
          :key="resource.id"
          :resource="resource"
        />
      </div>

      <KunNull v-else description="没有找到匹配的 Galgame 资源" />
    </KunLoading>

    <KunPagination
      v-if="total > LIMIT"
      v-model:current-page="page"
      :total-page="Math.ceil(total / LIMIT)"
      :is-loading="isPending"
    />
  </div>
</template>
