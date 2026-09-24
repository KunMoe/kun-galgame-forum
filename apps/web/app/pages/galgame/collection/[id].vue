<script setup lang="ts">
import type { CollectionAlias } from '#shared/utils/api/schemas'

const route = useRoute()
const router = useRouter()
const aliasId = computed(() => String((route.params as { id: string }).id))

const { data, status } = await useApi<CollectionAlias>(
  () => `collection-alias:${aliasId.value}`,
  (api, { signal }) =>
    api.GET('/collection-aliases/{alias_id}', {
      params: { path: { alias_id: aliasId.value } },
      signal
    })
)

if (data.value) {
  await router.replace({
    path: `/collection/${data.value.collection_id}`,
    query: route.query
  })
}

useKunDisableSeo('收藏夹')
</script>

<template>
  <div>
    <KunNull
      v-if="status !== 'pending' && !data"
      description="收藏夹不存在或你没有权限查看"
    />
  </div>
</template>
