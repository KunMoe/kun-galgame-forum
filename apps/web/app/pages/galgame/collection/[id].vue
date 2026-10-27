<script setup lang="ts">
import type { CollectionAlias } from '#shared/utils/api/schemas'

const route = useRoute()
const router = useRouter()
const aliasId = computed(() => String((route.params as { id: string }).id))

const { data, pending } = await useKunApi<CollectionAlias>(
  () => `collection-alias:${aliasId.value}`,
  (api, { signal }) =>
    api.GET('/collection-aliases/{alias_id}', {
      params: { path: { alias_id: aliasId.value } },
      signal
    }),
  { critical: true }
)

const open = (alias: CollectionAlias) =>
  router.replace({
    path: `/collection/${alias.collection_id}`,
    query: route.query
  })

if (import.meta.server) {
  if (data.value) {
    await open(data.value)
  }
} else {
  watch(
    data,
    (alias) => {
      if (alias) {
        open(alias)
      }
    },
    { immediate: true }
  )
}

useKunDisableSeo('收藏夹')
</script>

<template>
  <div>
    <KunNull
      v-if="!pending && !data"
      description="收藏夹不存在或你没有权限查看"
    />
  </div>
</template>
