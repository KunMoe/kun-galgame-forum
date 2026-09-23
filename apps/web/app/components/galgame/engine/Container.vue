<script setup lang="ts">
import type { Engine } from '#shared/utils/api/schemas'
import { engineItemOf } from '~/utils/galgame/entityCards'

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

const data = computed(() => engines.value?.map((e) => engineItemOf(e, nameOf)))
</script>

<template>
  <div class="space-y-6">
    <KunHeader
      name="Galgame 引擎资料库"
      description="Galgame 引擎资料库, 这里展示了世界上大多数常见的 Galgame 制作引擎, 例如 KRKR 引擎, YU-RIS 引擎, 椎名理绪引擎等"
    >
      <template #endContent>
        <span>
          {{ `总计 ${data?.length || 0} 个引擎` }}
        </span>
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

    <KunNull v-if="data && !data.length" />
  </div>
</template>
