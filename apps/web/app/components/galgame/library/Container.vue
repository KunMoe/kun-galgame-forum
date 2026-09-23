<script setup lang="ts">
import type { WorkPage } from '#shared/utils/api/schemas'

const { page, limit, query } = useLibraryWorksQuery()

const { data, status, problem } = await useApi<WorkPage>(
  () => `library-works:${JSON.stringify(query.value)}`,
  (api, { signal }) =>
    api.GET('/library-works', { params: { query: query.value }, signal })
)
watch(problem, (p) => {
  if (p) {
    reportProblem(p)
  }
})

const galgames = useWorkCards(() => data.value?.items)
const total = computed(() => data.value?.total ?? 0)

const releaseLabel = (galgame: GalgameCard) => {
  if (galgame.release_date_tba) {
    return '发售日期待定'
  }
  return galgame.release_date?.slice(0, 10) || '发售日期未知'
}
</script>

<template>
  <div class="flex flex-col gap-3">
    <div class="z-10">
      <KunHeader name="Galgame 信息资料数据库">
        <template #endContent>
          <GalgameLibraryNav
            :total="data ? total : null"
            :pending="status === 'pending'"
          />
        </template>

        <template #description>
          <p class="text-default-500">
            数据库收录的全部 Galgame, 无论本站是否有下载资源。想找下载请前往
            <KunLink to="/galgame">Galgame 资源资料库</KunLink>。
          </p>
        </template>
      </KunHeader>
    </div>

    <KunLoading :loading="status === 'pending'">
      <GalgameCard v-if="galgames.length" :galgames="galgames">
        <template #meta="{ galgame }">
          <p class="text-default-500 mt-1 text-xs">
            {{ releaseLabel(galgame) }}
          </p>
        </template>
      </GalgameCard>
      <KunNull v-else description="没有找到符合条件的 Galgame" />
    </KunLoading>

    <KunCard
      v-if="galgames.length"
      :is-hoverable="false"
      :is-transparent="false"
      content-class="gap-3"
    >
      <KunPagination
        v-model:current-page="page"
        :total-page="Math.ceil(total / limit)"
        :is-loading="status === 'pending'"
      />
    </KunCard>
  </div>
</template>
