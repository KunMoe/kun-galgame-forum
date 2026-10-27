<script setup lang="ts">
import type { FollowedCompanyList } from '#shared/utils/api/schemas'
import { problemMessage } from '#shared/utils/api/message'
import { companyItemOf } from '~/utils/galgame/entityCards'

const LIMIT = 24
const pageData = reactive({ page: usePageQuery(), limit: LIMIT })
const nameOf = useCatalogName()

const { data, status, pending, refresh, problem } =
  useKunApi<FollowedCompanyList>(
    () => `me-followed-companies:${pageData.page}:${pageData.limit}`,
    (api, { signal }) =>
      api.GET('/me/followed-companies', {
        params: {
          query: { page: pageData.page, limit: pageData.limit }
        },
        signal
      })
  )

const officials = computed(() =>
  (data.value?.items ?? []).map((item) => companyItemOf(item.company, nameOf))
)

const onFollowChange = async (delta: number) => {
  if (delta >= 0) {
    return
  }
  if ((data.value?.items.length ?? 0) <= 1 && pageData.page > 1) {
    pageData.page -= 1
    return
  }
  await refresh()
}
</script>

<template>
  <div class="space-y-3">
    <UserListSkeleton v-if="pending" shape="media" />
    <template v-else>
      <div v-if="officials.length" class="flex flex-col gap-3">
        <div
          v-for="official in officials"
          :key="official.id"
          class="flex flex-wrap items-center gap-3"
        >
          <div class="min-w-0 flex-1">
            <GalgameOfficialCard :official="official" />
          </div>
          <GalgameOfficialFollowButton
            :company-id="official.id"
            @change="onFollowChange"
          />
        </div>

        <KunPagination
          v-if="data && data.total > pageData.limit"
          v-model:current-page="pageData.page"
          :total-page="Math.ceil(data.total / pageData.limit)"
          :is-loading="status === 'pending'"
        />
      </div>

      <KunNull v-else-if="problem" :description="problemMessage(problem)" />

      <KunNull
        v-else-if="status !== 'pending'"
        description="还没有关注任何厂商"
      />
    </template>
  </div>
</template>
