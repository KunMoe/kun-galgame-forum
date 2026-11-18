<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type { EnvChange } from '#shared/utils/api/schemas'
import { KUN_ENV_CHANGE } from '~/constants/setting-env'
import { toKunUser } from '~/utils/userRef'

type ChangePage = { items: EnvChange[]; total: number }

const api = useApiClient()
const page = ref(1)
const limit = 20
const data = ref<ChangePage | null>(null)
const loading = ref(false)

const load = async () => {
  loading.value = true
  const result = await settle(
    api.GET('/admin/settings/env/changes', {
      params: { query: { page: page.value, limit } }
    })
  )
  loading.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  data.value = result.data
}

onMounted(load)
watch(page, load)

const totalPage = computed(() =>
  data.value ? Math.max(1, pageCountOf(data.value.total, limit)) : 1
)
</script>

<template>
  <div class="space-y-3">
    <KunLoading v-if="loading && !data" />
    <KunNull
      v-else-if="!data?.items.length"
      description="暂无配置中心变更记录"
    />

    <template v-else>
      <div
        v-for="entry in data.items"
        :key="entry.id"
        class="border-default-200 space-y-1 rounded-lg border p-3"
      >
        <div class="flex flex-wrap items-center gap-2">
          <KunChip
            size="sm"
            :color="entry.change_kind === 'delete' ? 'danger' : 'primary'"
          >
            {{ KUN_ENV_CHANGE[entry.change_kind] }}
          </KunChip>
          <span class="font-mono text-sm font-medium">{{
            entry.variable
          }}</span>
          <KunUserChip
            v-if="entry.changed_by"
            :user="toKunUser(entry.changed_by)"
            size="sm"
          />
          <span v-else class="text-default-400 text-xs">导入工具</span>
          <KunTime
            :time="entry.created_at"
            type="datetime"
            show-year
            class="ml-auto text-xs"
          />
        </div>
        <p class="font-mono text-xs break-all">
          <span class="text-default-500">{{
            entry.shown_before || '(无)'
          }}</span>
          →
          <span>{{ entry.shown_after || '(无)' }}</span>
        </p>
        <p v-if="entry.change_note" class="text-default-500 text-xs">
          {{ entry.change_note }}
        </p>
      </div>

      <KunPagination
        v-if="data.total > limit"
        v-model:current-page="page"
        :total-page="totalPage"
        :is-loading="loading"
      />
    </template>
  </div>
</template>
