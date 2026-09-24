<script setup lang="ts">
import type { PageListCollectionSummary } from '#shared/utils/api/schemas'
import { deletedUserName } from '#shared/utils/deletedUser'

const props = defineProps<{
  userId: number
  ownerName: string | null
}>()

const { allowsNsfw, stanceKey } = useContentStance()
const page = usePageQuery()
const limit = 24

const { data, status } = await useApi<PageListCollectionSummary>(
  () =>
    `user-collections:${props.userId}:${page.value}:${limit}:${stanceKey.value}`,
  (api, { signal }) =>
    api.GET('/users/{user_id}/collections', {
      params: {
        path: { user_id: String(props.userId) },
        query: {
          page: page.value,
          limit,
          include_nsfw: allowsNsfw.value
        }
      },
      signal
    })
)
</script>

<template>
  <div class="space-y-3">
    <div v-if="data && data.items.length" class="flex flex-col space-y-3">
      <div class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
        <GalgameCollectionCard
          v-for="c in data.items"
          :key="c.id"
          :collection="c"
          :owner-name="ownerName ?? deletedUserName"
        />
      </div>

      <KunPagination
        v-if="data.total > limit"
        v-model:current-page="page"
        :total-page="Math.ceil(data.total / limit)"
        :is-loading="status === 'pending'"
      />
    </div>

    <KunNull
      v-if="data && !data.items.length"
      description="这只笨蛋萝莉还没有任何收藏夹"
    />
  </div>
</template>
