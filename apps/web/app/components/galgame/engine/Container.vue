<script setup lang="ts">
import type { Engine } from '#shared/utils/api/schemas'
import { engineItemOf } from '~/utils/galgame/entityCards'
import { KUN_GALGAME_ENGINE_SORT_TABS } from '~/constants/galgame'

const nameOf = useCatalogName()

// Catalog records a couple of hundred engines; the page shows them all, which
// takes more than one 100-row page.
const { data: engines } = await useApi<Engine[]>('engines:all', async (api) => {
  const all: Engine[] = []
  for (let page = 1; page <= 10; page++) {
    const res = await api.GET('/engines', {
      params: { query: { page, limit: 100 } }
    })
    if (!res.data) {
      return res
    }
    all.push(...res.data.items)
    if (all.length >= res.data.total) {
      break
    }
  }
  return { data: all, response: new Response(null, { status: 200 }) }
})

const searchQuery = ref('')
const sort = ref('popularity')

const data = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  const items = engines.value ?? []
  const matched = q
    ? items.filter((e) => {
        const { name, original } = nameOf(e)
        return [name, original, ...e.aliases].some((text) =>
          text.toLowerCase().includes(q)
        )
      })
    : items
  const cards = matched.map((e) => engineItemOf(e, nameOf))
  return sort.value === 'popularity'
    ? cards.toSorted((a, b) => b.rater_count - a.rater_count || a.id - b.id)
    : cards
})
</script>

<template>
  <div class="space-y-6">
    <KunHeader
      name="Galgame 引擎资料库"
      description="Galgame 引擎资料库, 这里展示了世界上大多数常见的 Galgame 制作引擎, 例如 KRKR 引擎, YU-RIS 引擎, 椎名理绪引擎等"
    >
      <template #endContent>
        <div>
          <KunInput
            v-model="searchQuery"
            type="text"
            placeholder="搜索引擎名称或别名..."
          />
          <KunTab
            v-model="sort"
            :items="KUN_GALGAME_ENGINE_SORT_TABS"
            variant="bordered"
            size="sm"
            class-name="mt-4"
          />
          <p v-if="sort === 'popularity'" class="text-default-400 mt-2 text-xs">
            热度为引擎下作品在 VNDB、Bangumi、批评空间的评分人数之和
          </p>
          <div class="text-default-600 mt-4 text-sm">
            {{
              searchQuery.trim()
                ? `搜索结果: ${data.length} 个引擎`
                : `总计 ${data.length} 个引擎`
            }}
          </div>
        </div>
      </template>
    </KunHeader>

    <div
      class="grid grid-cols-2 gap-3 sm:grid-cols-2 sm:gap-3 lg:grid-cols-3 xl:grid-cols-4"
    >
      <GalgameEngineCard
        v-for="engine in data"
        :key="engine.id"
        :engine="engine"
      />
    </div>

    <KunNull v-if="!data.length" />
  </div>
</template>
