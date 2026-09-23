<script setup lang="ts">
import { watchDebounced } from '@vueuse/core'
import { useRouteQuery } from '@vueuse/router'
import type { CompanyPage } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'
import { companyItemOf } from '~/utils/galgame/entityCards'

const page = useRouteQuery('page', 1, { mode: 'replace', transform: Number })
const limit = 100
const api = useApiClient()
const nameOf = useCatalogName()

const { data, status } = await useApi<CompanyPage>(
  () => `companies:${page.value}`,
  (client) =>
    client.GET('/companies', { params: { query: { page: page.value, limit } } })
)

const searchResult = ref<GalgameOfficialItem[]>([])
const searchQuery = ref('')
const isSearching = ref(false)
const displayOfficials = computed(() =>
  searchQuery.value.trim()
    ? searchResult.value
    : (data.value?.items ?? []).map((c) => companyItemOf(c, nameOf))
)

const handleSearch = async () => {
  const q = searchQuery.value.trim()
  if (!q) {
    searchResult.value = []
    return
  }
  isSearching.value = true
  const res = await settle(
    api.GET('/companies', { params: { query: { q, limit: 100 } } })
  )
  isSearching.value = false
  searchResult.value = res.ok
    ? res.data.items.map((c) => companyItemOf(c, nameOf))
    : []
}

watchDebounced(
  () => searchQuery.value,
  () => {
    handleSearch()
  },
  { debounce: 500, maxWait: 1000 }
)
</script>

<template>
  <div class="space-y-6">
    <KunHeader
      name="Galgame 会社 / 厂商资料库"
      description="这里展示了绝大多数 Galgame 的制作厂商 / Galgame 会社, 并有会社别名 (例如 Yuzusoft 的别名为柚子社), 按作品数量从多到少排列, 您可以点击会社以查看这个会社制作的所有 Galgame"
    >
      <template #endContent>
        <div>
          <KunInput
            v-model="searchQuery"
            type="text"
            placeholder="搜索会社名称、描述或别名..."
          />

          <div class="text-default-600 mt-4 flex items-center gap-4 text-sm">
            <span v-if="!searchQuery.trim()">
              {{ `总计 ${data?.total || 0} 个会社` }}
            </span>
            <span v-else>{{ `搜索结果: ${searchResult.length} 个会社` }}</span>
          </div>
        </div>
      </template>
    </KunHeader>

    <div
      class="grid grid-cols-2 gap-3 sm:grid-cols-2 sm:gap-3 lg:grid-cols-3 xl:grid-cols-4"
    >
      <GalgameOfficialCard
        v-for="official in displayOfficials"
        :key="official.id"
        :official="official"
      />
    </div>

    <KunNull
      v-if="!isSearching && !displayOfficials.length && searchQuery.trim()"
    />

    <KunLoading v-if="isSearching" />

    <KunPagination
      v-if="data && data.total > limit && !searchQuery.trim()"
      v-model:current-page="page"
      :total-page="Math.ceil(data.total / limit)"
      :is-loading="status === 'pending'"
    />
  </div>
</template>
